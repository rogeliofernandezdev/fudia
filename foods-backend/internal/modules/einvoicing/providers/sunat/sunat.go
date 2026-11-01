// Package sunat is Fudia's own issuing engine for Peru: it builds the UBL 2.1
// XML, signs it with the company's certificate, sends it to SUNAT (sendBill)
// and reads the CDR. Beta works with SUNAT's public test credentials and a
// self-signed certificate generated in memory.
package sunat

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

const (
	EndpointBeta       = "https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService"
	EndpointProduction = "https://e-factura.sunat.gob.pe/ol-ti-itcpfegem/billService"
	betaUser           = "MODDATOS"
	betaPassword       = "MODDATOS"

	FieldSOLUser             = "solUser"
	FieldSOLPassword         = "solPassword"
	FieldCertificate         = "certificate"
	FieldCertificatePassword = "certificatePassword"
)

var solUserPattern = regexp.MustCompile(`^[A-Za-z0-9]{1,20}$`)

type Provider struct {
	client    *http.Client
	endpoints map[string]string
	testOnce  sync.Once
	testCert  signer
	testErr   error
}

var _ ports.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{
		client:    &http.Client{Timeout: 45 * time.Second},
		endpoints: map[string]string{domain.EnvironmentTest: EndpointBeta, domain.EnvironmentProduction: EndpointProduction},
	}
}

// WithEndpoints overrides SUNAT URLs (tests use a local server).
func (p *Provider) WithEndpoints(test, production string) *Provider {
	p.endpoints = map[string]string{domain.EnvironmentTest: test, domain.EnvironmentProduction: production}
	return p
}

func (p *Provider) Key() string    { return "sunat" }
func (p *Provider) Label() string  { return "SUNAT directo (firma propia)" }
func (p *Provider) TestOnly() bool { return false }
func (p *Provider) CredentialFields() []ports.CredentialField {
	return []ports.CredentialField{
		{Key: FieldSOLUser, Label: "Usuario SOL secundario", Type: "text", Required: true},
		{Key: FieldSOLPassword, Label: "Clave SOL", Type: "secret", Required: true},
		{Key: FieldCertificate, Label: "Certificado digital (.pfx)", Type: "file", Accept: ".pfx,.p12", Required: true},
		{Key: FieldCertificatePassword, Label: "Contraseña del certificado", Type: "secret", Required: true},
	}
}

// Beta accepts SUNAT's public test user and any certificate.
func (p *Provider) RequiresCredentials(environment string) bool {
	return environment == domain.EnvironmentProduction
}

func (p *Provider) ValidateCredentials(environment string, values map[string]string) (map[string]string, error) {
	meta := map[string]string{}
	user, password := strings.TrimSpace(values[FieldSOLUser]), values[FieldSOLPassword]
	if environment == domain.EnvironmentProduction && (user == "" || password == "") {
		return nil, &ports.CredentialError{Message: "Ingresa el usuario y la clave SOL secundarios."}
	}
	if user != "" {
		if !solUserPattern.MatchString(user) {
			return nil, &ports.CredentialError{Message: "El usuario SOL solo admite letras y números."}
		}
		meta[FieldSOLUser] = strings.ToUpper(user)
	}
	if values[FieldCertificate] == "" {
		if environment == domain.EnvironmentProduction {
			return nil, &ports.CredentialError{Message: "Carga el certificado digital de la empresa."}
		}
		return meta, nil
	}
	s, err := decodeCertificate(values[FieldCertificate], values[FieldCertificatePassword])
	if err != nil {
		return nil, err
	}
	if time.Now().After(s.cert.NotAfter) {
		return nil, &ports.CredentialError{Message: "El certificado digital está vencido."}
	}
	meta["certificateSubject"] = s.cert.Subject.CommonName
	meta["certificateIssuer"] = s.cert.Issuer.CommonName
	meta["certificateExpiresAt"] = s.cert.NotAfter.UTC().Format(time.RFC3339)
	return meta, nil
}

func decodeCertificate(encoded, password string) (signer, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return signer{}, &ports.CredentialError{Message: "El certificado no es un archivo válido."}
	}
	key, cert, _, err := pkcs12.DecodeChain(raw, password)
	if err != nil {
		return signer{}, &ports.CredentialError{Message: "No pudimos abrir el certificado: revisa el archivo y su contraseña."}
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return signer{}, &ports.CredentialError{Message: "El certificado debe tener una llave RSA."}
	}
	return signer{key: rsaKey, cert: cert}, nil
}

// testSigner is a self-signed certificate for SUNAT beta, created once.
func (p *Provider) testSigner() (signer, error) {
	p.testOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			p.testErr = err
			return
		}
		template := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "Fudia certificado de pruebas"},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(1, 0, 0), KeyUsage: x509.KeyUsageDigitalSignature}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			p.testErr = err
			return
		}
		cert, err := x509.ParseCertificate(der)
		p.testCert, p.testErr = signer{key: key, cert: cert}, err
	})
	return p.testCert, p.testErr
}

func (p *Provider) credentials(d ports.Document) (string, string, signer, error) {
	user, password := strings.TrimSpace(d.Credentials[FieldSOLUser]), d.Credentials[FieldSOLPassword]
	production := d.Environment == domain.EnvironmentProduction
	if user == "" && !production {
		user, password = betaUser, betaPassword
	}
	if user == "" || password == "" {
		return "", "", signer{}, &ports.CredentialError{Message: "Faltan el usuario y la clave SOL."}
	}
	username := strings.ToUpper(user)
	if !strings.HasPrefix(username, d.Issuer.TaxID) {
		username = d.Issuer.TaxID + username
	}
	if d.Credentials[FieldCertificate] != "" {
		s, err := decodeCertificate(d.Credentials[FieldCertificate], d.Credentials[FieldCertificatePassword])
		return username, password, s, err
	}
	if production {
		return "", "", signer{}, &ports.CredentialError{Message: "Falta el certificado digital."}
	}
	s, err := p.testSigner()
	return username, password, s, err
}

// Issue signs and sends one factura or boleta. Rejections (SUNAT codes
// 2000-3999) are final; connection problems and other SUNAT exceptions are
// returned as errors so the document is retried after they are fixed.
func (p *Provider) Issue(ctx context.Context, d ports.Document) (ports.Result, error) {
	if d.FiscalCode != "01" && d.FiscalCode != "03" {
		return ports.Result{}, fmt.Errorf("tipo de comprobante %s no soportado", d.FiscalCode)
	}
	endpoint := p.endpoints[d.Environment]
	if endpoint == "" {
		return ports.Result{}, fmt.Errorf("ambiente %s sin endpoint de SUNAT", d.Environment)
	}
	username, password, sign, err := p.credentials(d)
	if err != nil {
		return ports.Result{}, err
	}
	doc := buildInvoice(d)
	digest, err := sign.sign(doc)
	if err != nil {
		return ports.Result{}, fmt.Errorf("no se pudo firmar el comprobante: %w", err)
	}
	signed, err := doc.WriteToBytes()
	if err != nil {
		return ports.Result{}, err
	}
	name := fmt.Sprintf("%s-%s-%s", d.Issuer.TaxID, d.FiscalCode, d.FormattedNumber)
	zipped, err := zipFile(name+".xml", signed)
	if err != nil {
		return ports.Result{}, err
	}
	result := ports.Result{DigestValue: digest, Files: []ports.File{{Kind: "xml", Name: name + ".xml", ContentType: "application/xml", Content: signed}}}
	response, err := sendBill(ctx, p.client, endpoint, username, password, name+".zip", zipped)
	var f *fault
	if errors.As(err, &f) {
		switch {
		case f.Code >= 2000 && f.Code < 4000, f.Code == 1033:
			result.Status, result.Code, result.Message = domain.StatusRejected, fmt.Sprintf("%04d", f.Code), f.Message
			return result, nil
		default:
			return result, f
		}
	}
	if err != nil {
		return result, err
	}
	receipt, err := readCDR(response)
	if err != nil {
		return result, fmt.Errorf("constancia de SUNAT ilegible: %w", err)
	}
	result.Files = append(result.Files, ports.File{Kind: "cdr", Name: "R-" + name + ".zip", ContentType: "application/zip", Content: response})
	result.Code, result.Message = receipt.ResponseCode, receipt.Description
	switch {
	case receipt.ResponseCode == "0" && len(receipt.Notes) > 0:
		result.Status, result.Message = domain.StatusObserved, receipt.Description+" Observaciones: "+strings.Join(receipt.Notes, "; ")
	case receipt.ResponseCode == "0":
		result.Status = domain.StatusAccepted
	default:
		result.Status = domain.StatusRejected
	}
	result.Reference = name
	return result, nil
}
