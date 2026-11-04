package sunat

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	dsig "github.com/russellhaering/goxmldsig"
)

func TestInvoiceXMLAndSignatureVerify(t *testing.T) {
	p := New()
	s, err := p.testSigner()
	if err != nil {
		t.Fatal(err)
	}
	mype := []domain.TaxComponent{{Code: "igv", Name: "IGV", Rate: "0.080000"}, {Code: "ipm", Name: "IPM", Rate: "0.025000"}}
	d := sampleDocument(t, "01", "F001", 7, domain.Customer{DocumentType: "RUC", DocumentNumber: "20100047218", Name: "Empresa & Cía"}, "6", mype)
	doc := buildInvoice(d)
	digest, err := s.sign(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := doc.WriteToBytes()
	xml := string(raw)
	for _, want := range []string{`<cbc:ID>F001-7</cbc:ID>`, `listID="0101">01</cbc:InvoiceTypeCode>`, `<cbc:Percent>10.5</cbc:Percent>`,
		`<cbc:PaymentMeansID>Contado</cbc:PaymentMeansID>`, `languageLocaleID="1000">OCHENTA Y CINCO CON 30/100 SOLES</cbc:Note>`,
		`unitCode="ZZ">1</cbc:InvoicedQuantity>`, `Empresa &amp; Cía`, `<ds:Signature Id="SignFUDIA">`} {
		if !strings.Contains(xml, want) {
			t.Errorf("XML without %s", want)
		}
	}

	// Verify as SUNAT does: remove the signature (enveloped), canonicalize,
	// compare digests, then check SignedInfo with root namespaces in scope.
	parsed := etree.NewDocument()
	if err := parsed.ReadFromBytes(raw); err != nil {
		t.Fatal(err)
	}
	signature := parsed.FindElement("//ds:Signature")
	signedInfo := signature.SelectElement("ds:SignedInfo").Copy()
	value, _ := base64.StdEncoding.DecodeString(signature.SelectElement("ds:SignatureValue").Text())
	signature.Parent().RemoveChild(signature)
	canonical, err := dsig.MakeC14N10RecCanonicalizer().Canonicalize(parsed.Root())
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum(canonical)
	if base64.StdEncoding.EncodeToString(sum[:]) != digest {
		t.Fatal("document digest does not verify")
	}
	for _, ns := range rootNamespaces {
		signedInfo.CreateAttr(ns[0], ns[1])
	}
	canonicalInfo, _ := dsig.MakeC14N10RecCanonicalizer().Canonicalize(signedInfo)
	infoSum := sha1.Sum(canonicalInfo)
	if err := rsa.VerifyPKCS1v15(&s.key.PublicKey, crypto.SHA1, infoSum[:], value); err != nil {
		t.Fatalf("signature does not verify: %v", err)
	}
}

func cdrZip(t *testing.T, code, description string, notes ...string) []byte {
	t.Helper()
	var noteXML string
	for _, n := range notes {
		noteXML += `<cbc:Note>` + n + `</cbc:Note>`
	}
	content := `<?xml version="1.0"?><ar:ApplicationResponse xmlns:ar="urn:oasis:names:specification:ubl:schema:xsd:ApplicationResponse-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">` +
		noteXML + `<cac:DocumentResponse><cac:Response><cbc:ResponseCode>` + code + `</cbc:ResponseCode><cbc:Description>` + description + `</cbc:Description></cac:Response></cac:DocumentResponse></ar:ApplicationResponse>`
	zipped, err := zipFile("R-test.xml", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return zipped
}

func soapServer(t *testing.T, status int, body func(*http.Request) string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body(r))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestIssueReadsCDRFaultsAndConnectionErrors(t *testing.T) {
	general := []domain.TaxComponent{{Code: "general", Name: "IGV", Rate: "0.180000"}}
	d := sampleDocument(t, "03", "B001", 9, domain.Customer{}, "0", general)
	var seenUser, seenFile string
	accepted := soapServer(t, 200, func(r *http.Request) string {
		envelope, _ := io.ReadAll(r.Body)
		doc := etree.NewDocument()
		_ = doc.ReadFromBytes(envelope)
		seenUser, seenFile = doc.FindElement("//Username").Text(), doc.FindElement("//fileName").Text()
		zipped, _ := base64.StdEncoding.DecodeString(doc.FindElement("//contentFile").Text())
		reader, _ := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
		if len(reader.File) != 1 || reader.File[0].Name != "20000000001-03-B001-9.xml" {
			t.Errorf("zip entry: %+v", reader.File)
		}
		return `<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"><soap-env:Body><br:sendBillResponse xmlns:br="http://service.sunat.gob.pe"><applicationResponse>` +
			base64.StdEncoding.EncodeToString(cdrZip(t, "0", "La Boleta numero B001-9, ha sido aceptada", "4252 - observación de prueba")) + `</applicationResponse></br:sendBillResponse></soap-env:Body></soap-env:Envelope>`
	})
	result, err := New().WithEndpoints(accepted.URL, accepted.URL).Issue(context.Background(), d)
	if err != nil || result.Status != domain.StatusObserved || !strings.Contains(result.Message, "4252") || len(result.Files) != 2 || result.DigestValue == "" {
		t.Fatalf("observed CDR: %+v %v", result, err)
	}
	if seenUser != "20000000001MODDATOS" || seenFile != "20000000001-03-B001-9.zip" {
		t.Fatalf("beta credentials and file name: %s %s", seenUser, seenFile)
	}

	rejected := soapServer(t, 500, func(*http.Request) string {
		return `<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"><soap-env:Body><soap-env:Fault><faultcode>soap-env:Client.2017</faultcode><faultstring>El numero de documento de identidad del receptor debe ser RUC</faultstring></soap-env:Fault></soap-env:Body></soap-env:Envelope>`
	})
	result, err = New().WithEndpoints(rejected.URL, rejected.URL).Issue(context.Background(), d)
	if err != nil || result.Status != domain.StatusRejected || result.Code != "2017" || len(result.Files) != 1 {
		t.Fatalf("rejection: %+v %v", result, err)
	}

	unauthorized := soapServer(t, 500, func(*http.Request) string {
		return `<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"><soap-env:Body><soap-env:Fault><faultcode>soap-env:Client.0102</faultcode><faultstring>Usuario o contraseña incorrectos</faultstring></soap-env:Fault></soap-env:Body></soap-env:Envelope>`
	})
	if _, err = New().WithEndpoints(unauthorized.URL, unauthorized.URL).Issue(context.Background(), d); err == nil || !strings.Contains(err.Error(), "0102") {
		t.Fatalf("a configuration error must be retryable: %v", err)
	}
	throttled := soapServer(t, 401, func(*http.Request) string { return "" })
	if _, err = New().WithEndpoints(throttled.URL, throttled.URL).Issue(context.Background(), d); err == nil {
		t.Fatal("an HTTP error without XML must be retryable")
	}

	production := d
	production.Environment = domain.EnvironmentProduction
	if _, err := New().Issue(context.Background(), production); err == nil {
		t.Fatal("production without credentials must not send")
	}
}

func TestValidateCredentials(t *testing.T) {
	p := New()
	if _, err := p.ValidateCredentials(domain.EnvironmentProduction, map[string]string{}); err == nil {
		t.Fatal("production requires SOL user and certificate")
	}
	meta, err := p.ValidateCredentials(domain.EnvironmentTest, map[string]string{FieldSOLUser: "factura1", FieldSOLPassword: "x"})
	if err != nil || meta[FieldSOLUser] != "FACTURA1" {
		t.Fatalf("beta credentials: %v %v", meta, err)
	}
	if _, err := p.ValidateCredentials(domain.EnvironmentTest, map[string]string{FieldSOLUser: "bad user"}); err == nil {
		t.Fatal("invalid SOL user")
	}
	if _, err := p.ValidateCredentials(domain.EnvironmentTest, map[string]string{FieldCertificate: base64.StdEncoding.EncodeToString([]byte("not a pfx")), FieldCertificatePassword: "x"}); err == nil {
		t.Fatal("an invalid certificate must be rejected")
	}
	if !p.RequiresCredentials(domain.EnvironmentProduction) || p.RequiresCredentials(domain.EnvironmentTest) {
		t.Fatal("credential requirements by environment")
	}
	var _ ports.Provider = p
}
