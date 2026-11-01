package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
)

// testRUC returns a unique RUC with a valid check digit.
func testRUC() string {
	base := fmt.Sprintf("20%08d", time.Now().UnixNano()%100000000)
	weights := []int{5, 4, 3, 2, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i, w := range weights {
		sum += int(base[i]-'0') * w
	}
	check := 11 - sum%11
	if check == 10 {
		check = 0
	} else if check == 11 {
		check = 1
	}
	return fmt.Sprintf("%s%d", base, check)
}

// Checkout → fiscal document → simulated provider, with the document stored in
// the payment transaction: a rejected request leaves no payment behind.
func TestCheckoutIssuesElectronicReceiptsAndInvoices(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	grantCashManagementForPaymentTest(t, api, s)
	e := api.handlers.EInvoicing

	call := func(handler http.HandlerFunc, method, body string, path map[string]string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/test", bytes.NewReader([]byte(body)))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		for key, value := range path {
			r.SetPathValue(key, value)
		}
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, body, want, w.Code, w.Body.String())
		}
		return w
	}
	code := func(w *httptest.ResponseRecorder) string {
		var body struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return body.Code
	}
	var registerID, profileID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja comprobantes',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	call(api.openCashShift, "POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`, registerID), nil, 201)
	if err := pool.QueryRow(ctx, `SELECT fiscal_profile_id FROM locations WHERE id=$1`, s.LocationID).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	order := func(items ...[3]string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,created_by) VALUES($1,$2,$3,'mostrador','listo',$4) RETURNING id`,
			s.OrganizationID, s.LocationID, fmt.Sprintf("PED-FE-%d", time.Now().UnixNano()), s.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			var productID string
			if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,$2,$3,$4::numeric) RETURNING id`,
				s.OrganizationID, fmt.Sprintf("FE-%d", time.Now().UnixNano()), item[0], item[2]).Scan(&productID); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price) VALUES($1,$2,$3,$4,$5::numeric,$6::numeric)`, s.OrganizationID, id, productID, item[0], item[1], item[2]); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := pool.Exec(ctx, `UPDATE orders SET subtotal=(SELECT sum(qty*unit_price) FROM order_items WHERE order_id=$1),total=(SELECT sum(qty*unit_price) FROM order_items WHERE order_id=$1) WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	pay := func(orderID, amount, fiscal string, want int) *httptest.ResponseRecorder {
		t.Helper()
		body := fmt.Sprintf(`{"orderId":%q,"method":"card","amount":%s,"reference":"VISA"`, orderID, amount)
		if fiscal != "" {
			body += `,"fiscal":` + fiscal
		}
		return call(api.createPayment, "POST", body+"}", nil, want)
	}
	documentOf := func(w *httptest.ResponseRecorder) (string, string) {
		t.Helper()
		var out struct {
			FiscalDocument struct {
				ID     string `json:"id"`
				Number string `json:"number"`
			} `json:"fiscalDocument"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.FiscalDocument.ID == "" {
			t.Fatalf("payment without fiscal document: %s", w.Body.String())
		}
		return out.FiscalDocument.ID, out.FiscalDocument.Number
	}
	status := func(id string) (string, int) {
		t.Helper()
		var st string
		var attempts int
		if err := pool.QueryRow(ctx, `SELECT status,attempts FROM fiscal_documents WHERE id=$1`, id).Scan(&st, &attempts); err != nil {
			t.Fatal(err)
		}
		return st, attempts
	}

	// Without the module nothing is offered and enabling is refused.
	var options struct {
		Enabled           bool   `json:"enabled"`
		DefaultDocument   string `json:"defaultDocument"`
		IdentityThreshold string `json:"identityThreshold"`
		DocumentTypes     []struct {
			Value string `json:"value"`
		} `json:"documentTypes"`
	}
	_ = json.Unmarshal(call(e.CheckoutOptions, "GET", "", nil, 200).Body.Bytes(), &options)
	if options.Enabled {
		t.Fatal("checkout must not offer documents without the module")
	}
	enable := `{"enabled":true,"provider":"fake","environment":"test"}`
	if w := call(e.SaveProfile, "PUT", enable, map[string]string{"id": profileID}, 409); code(w) != "einvoicing_not_entitled" {
		t.Fatalf("not entitled: %s", w.Body.String())
	}
	if w := pay(order([3]string{"Café", "1", "6.00"}), "6", `{"documentType":"receipt"}`, 409); code(w) != "einvoicing_not_enabled" {
		t.Fatalf("receipt without module: %s", w.Body.String())
	}

	if _, err := pool.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'facturacion',true)
		ON CONFLICT (organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if w := call(e.SaveProfile, "PUT", enable, map[string]string{"id": profileID}, 409); code(w) != "einvoicing_issuer_incomplete" {
		t.Fatalf("an invalid issuer RUC must block enabling: %s", w.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE organizations SET tax_id=$2 WHERE id=$1`, s.OrganizationID, testRUC()); err != nil {
		t.Fatal(err)
	}
	call(e.SaveProfile, "PUT", `{"enabled":true,"provider":"fake","environment":"production"}`, map[string]string{"id": profileID}, 400)
	call(e.SaveProfile, "PUT", enable, map[string]string{"id": profileID}, 200)
	call(e.SaveLocation, "PUT", `{"defaultDocument":"receipt","series":{"receipt":"F001","invoice":"F001"}}`, map[string]string{"id": s.LocationID}, 400)
	call(e.SaveLocation, "PUT", `{"defaultDocument":"receipt","series":{"receipt":"B001","invoice":"F001"}}`, map[string]string{"id": s.LocationID}, 200)
	_ = json.Unmarshal(call(e.CheckoutOptions, "GET", "", nil, 200).Body.Bytes(), &options)
	if !options.Enabled || options.DefaultDocument != "receipt" || len(options.DocumentTypes) != 3 || options.IdentityThreshold != "700.00" {
		t.Fatalf("checkout options: %+v", options)
	}

	// An anonymous receipt: stored with the payment, sent and accepted.
	first := order([3]string{"Lomo saltado", "2", "35.90"}, [3]string{"Chicha morada", "1", "8.50"})
	receiptID, receiptNumber := documentOf(pay(first, "80.30", `{"documentType":"receipt"}`, 201))
	if st, _ := status(receiptID); receiptNumber != "B001-1" || st != "ACCEPTED" {
		t.Fatalf("receipt: %s %s", receiptNumber, st)
	}
	var net, tax, total string
	if err := pool.QueryRow(ctx, `SELECT net_total::text,tax_total::text,total::text FROM fiscal_documents WHERE id=$1`, receiptID).Scan(&net, &tax, &total); err != nil {
		t.Fatal(err)
	}
	if net != "68.05" || tax != "12.25" || total != "80.30" {
		t.Fatalf("IGV 18%% breakdown: net %s tax %s total %s", net, tax, total)
	}

	// Invalid invoice data rolls the payment back and does not consume a number.
	second := order([3]string{"Ceviche", "1", "42.00"})
	if w := pay(second, "42", `{"documentType":"invoice","customer":{"documentType":"RUC","documentNumber":"20100047219","name":"Empresa SAC"}}`, 400); code(w) != "invalid_fiscal_customer" {
		t.Fatalf("invalid RUC: %s", w.Body.String())
	}
	if w := pay(second, "20", `{"documentType":"invoice","customer":{"documentType":"RUC","documentNumber":"20100047218","name":"Empresa SAC"}}`, 409); code(w) != "fiscal_requires_full_payment" {
		t.Fatalf("partial payment: %s", w.Body.String())
	}
	var payments int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1`, second).Scan(&payments); err != nil || payments != 0 {
		t.Fatalf("rejected fiscal requests must not leave payments: %d %v", payments, err)
	}
	invoiceID, invoiceNumber := documentOf(pay(second, "42", `{"documentType":"invoice","customer":{"documentType":"RUC","documentNumber":"20100047218","name":"Empresa SAC","address":"Av. Larco 123"}}`, 201))
	if st, _ := status(invoiceID); invoiceNumber != "F001-1" || st != "ACCEPTED" {
		t.Fatalf("invoice: %s %s", invoiceNumber, st)
	}

	// A temporary provider failure stays retryable; a rejection is final.
	failing := order([3]string{"Pisco sour", "1", "18.00"})
	errorID, _ := documentOf(pay(failing, "18", `{"documentType":"receipt","customer":{"documentType":"DNI","documentNumber":"45678901","name":"Ana [error]"}}`, 201))
	if st, attempts := status(errorID); st != "ERROR" || attempts != 1 {
		t.Fatalf("temporary failure: %s %d", st, attempts)
	}
	call(e.Retry, "POST", "", map[string]string{"id": errorID}, 200)
	if st, attempts := status(errorID); st != "ERROR" || attempts != 2 {
		t.Fatalf("retry must send again: %s %d", st, attempts)
	}
	rejectedID, _ := documentOf(pay(order([3]string{"Causa", "1", "15.00"}), "15", `{"documentType":"receipt","customer":{"name":"[rechazar]"}}`, 201))
	if st, _ := status(rejectedID); st != "REJECTED" {
		t.Fatalf("rejection: %s", st)
	}
	call(e.Retry, "POST", "", map[string]string{"id": rejectedID}, 409)

	// A payment without a document behaves as before.
	if w := pay(order([3]string{"Agua", "1", "4.00"}), "4", `{"documentType":"none"}`, 201); bytes.Contains(w.Body.Bytes(), []byte("fiscalDocument")) {
		t.Fatalf("no document requested: %s", w.Body.String())
	}

	// A dated tax regime applies its components to new documents.
	call(api.handlers.TaxRegimes.Create, "POST", fmt.Sprintf(`{"name":"MYPE","validFrom":"%s","components":[{"code":"igv","name":"IGV","rate":"0.08"},{"code":"ipm","name":"IPM","rate":"0.025"}]}`,
		time.Now().AddDate(0, 0, -2).Format("2006-01-02")), map[string]string{"id": profileID}, 201)
	mypeID, _ := documentOf(pay(order([3]string{"Menú", "1", "22.10"}), "22.10", `{"documentType":"receipt"}`, 201))
	var components int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fiscal_document_taxes WHERE document_id=$1 AND code IN ('igv','ipm')`, mypeID).Scan(&components); err != nil || components != 2 {
		t.Fatalf("regime components: %d %v", components, err)
	}

	// Listing, filters and detail of the session location.
	var list struct {
		Items []struct {
			Number string `json:"number"`
			Label  string `json:"label"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(call(e.List, "GET", "", nil, 200).Body.Bytes(), &list)
	if list.Total != 5 || list.Items[0].Number != "B001-4" {
		t.Fatalf("list: %+v", list)
	}
	r := httptest.NewRequest("GET", "/v1/admin/fiscal-documents?status=ACCEPTED", nil)
	r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
	w := httptest.NewRecorder()
	e.List(w, r)
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if list.Total != 3 {
		t.Fatalf("accepted filter: %+v", list)
	}
	var detail struct {
		Label     string `json:"label"`
		Breakdown struct {
			Lines []struct {
				Description string `json:"description"`
			} `json:"lines"`
			Taxes []struct {
				Code string `json:"code"`
			} `json:"taxes"`
		} `json:"breakdown"`
	}
	_ = json.Unmarshal(call(e.Get, "GET", "", map[string]string{"id": receiptID}, 200).Body.Bytes(), &detail)
	if detail.Label != "Boleta" || len(detail.Breakdown.Lines) != 2 || len(detail.Breakdown.Taxes) != 1 {
		t.Fatalf("detail: %+v", detail)
	}
	other := seedInventoryScope(t, pool)
	r = httptest.NewRequest("GET", "/x", nil)
	r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, other))
	r.SetPathValue("id", receiptID)
	w = httptest.NewRecorder()
	e.Get(w, r)
	if w.Code != 404 {
		t.Fatalf("another company must not read the document: %d", w.Code)
	}

	// A series that already issued documents cannot be renamed.
	if w := call(e.SaveLocation, "PUT", `{"defaultDocument":"none","series":{"receipt":"B002","invoice":"F001"}}`, map[string]string{"id": s.LocationID}, 409); code(w) != "einvoicing_series_in_use" {
		t.Fatalf("series in use: %s", w.Body.String())
	}
}

// The own SUNAT engine end to end against a local SOAP server: encrypted
// credentials, decrypted only for the send, signed XML and CDR stored.
func TestCheckoutSendsToSUNATWithEncryptedCredentials(t *testing.T) {
	var seenUser, seenPassword, seenXML string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		envelope, _ := io.ReadAll(r.Body)
		doc := etree.NewDocument()
		_ = doc.ReadFromBytes(envelope)
		seenUser, seenPassword = doc.FindElement("//Username").Text(), doc.FindElement("//Password").Text()
		zipped, _ := base64.StdEncoding.DecodeString(doc.FindElement("//contentFile").Text())
		reader, _ := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
		rc, _ := reader.File[0].Open()
		content, _ := io.ReadAll(rc)
		seenXML = string(content)
		var cdr bytes.Buffer
		zw := zip.NewWriter(&cdr)
		f, _ := zw.Create("R-cdr.xml")
		_, _ = f.Write([]byte(`<ar:ApplicationResponse xmlns:ar="urn:x" xmlns:cbc="urn:cbc"><cbc:ResponseCode>0</cbc:ResponseCode><cbc:Description>La Boleta ha sido aceptada</cbc:Description></ar:ApplicationResponse>`))
		_ = zw.Close()
		_, _ = io.WriteString(w, `<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"><soap-env:Body><br:sendBillResponse xmlns:br="http://service.sunat.gob.pe"><applicationResponse>`+base64.StdEncoding.EncodeToString(cdr.Bytes())+`</applicationResponse></br:sendBillResponse></soap-env:Body></soap-env:Envelope>`)
	}))
	defer server.Close()
	t.Setenv("FUDIA_TEST_SUNAT_URL", server.URL)

	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	grantCashManagementForPaymentTest(t, api, s)
	e := api.handlers.EInvoicing
	call := func(handler http.HandlerFunc, method, body string, path map[string]string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/test", bytes.NewReader([]byte(body)))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		for key, value := range path {
			r.SetPathValue(key, value)
		}
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, body, want, w.Code, w.Body.String())
		}
		return w
	}
	ruc := testRUC()
	var profileID, registerID, orderID, productID string
	if _, err := pool.Exec(ctx, `UPDATE organizations SET tax_id=$2,legal_name='Restaurante Fudia S.A.C.' WHERE id=$1`, s.OrganizationID, ruc); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'facturacion',true) ON CONFLICT (organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT fiscal_profile_id FROM locations WHERE id=$1`, s.LocationID).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja SUNAT',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	call(api.openCashShift, "POST", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`, registerID), nil, 201)

	// Beta works without stored credentials; production does not.
	call(e.SaveProfile, "PUT", `{"enabled":true,"provider":"sunat","environment":"production"}`, map[string]string{"id": profileID}, 409)
	call(e.SaveProfile, "PUT", `{"enabled":true,"provider":"sunat","environment":"test"}`, map[string]string{"id": profileID}, 200)
	call(e.SaveLocation, "PUT", `{"defaultDocument":"receipt","series":{"receipt":"B001","invoice":"F001"}}`, map[string]string{"id": s.LocationID}, 200)

	// Credentials are stored encrypted and never returned.
	call(e.SaveCredentials, "PUT", `{"provider":"sunat","values":{"solUser":"fact01","solPassword":"clave-secreta-123"}}`, map[string]string{"id": profileID}, 200)
	var ciphertext []byte
	if err := pool.QueryRow(ctx, `SELECT ciphertext FROM einvoicing_credentials WHERE fiscal_profile_id=$1 AND provider='sunat'`, profileID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("clave-secreta-123")) || bytes.Contains(ciphertext, []byte("fact01")) {
		t.Fatal("credentials must be stored encrypted")
	}
	view := call(e.Settings, "GET", "", nil, 200).Body.String()
	if strings.Contains(view, "clave-secreta-123") || !strings.Contains(view, `"solUser":"FACT01"`) || !strings.Contains(view, `"configured":true`) {
		t.Fatalf("settings must show metadata, never secrets: %s", view)
	}
	call(e.SaveCredentials, "PUT", `{"provider":"sunat","values":{"desconocido":"x"}}`, map[string]string{"id": profileID}, 400)
	if w := call(e.SaveProfile, "PUT", `{"enabled":true,"provider":"sunat","environment":"production"}`, map[string]string{"id": profileID}, 409); !strings.Contains(w.Body.String(), "certificado") {
		t.Fatalf("production needs a certificate: %s", w.Body.String())
	}

	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,$2,'Aji de gallina',32) RETURNING id`, s.OrganizationID, fmt.Sprintf("SUNAT-%d", time.Now().UnixNano())).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,subtotal,total,created_by) VALUES($1,$2,$3,'mostrador','listo',32,32,$4) RETURNING id`,
		s.OrganizationID, s.LocationID, fmt.Sprintf("PED-SUNAT-%d", time.Now().UnixNano()), s.UserID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price) VALUES($1,$2,$3,'Aji de gallina',1,32)`, s.OrganizationID, orderID, productID); err != nil {
		t.Fatal(err)
	}
	w := call(api.createPayment, "POST", fmt.Sprintf(`{"orderId":%q,"method":"card","amount":32,"reference":"VISA","fiscal":{"documentType":"receipt"}}`, orderID), nil, 201)
	var paid struct {
		FiscalDocument struct {
			ID string `json:"id"`
		} `json:"fiscalDocument"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &paid)
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM fiscal_documents WHERE id=$1`, paid.FiscalDocument.ID).Scan(&status); err != nil || status != "ACCEPTED" {
		t.Fatalf("SUNAT CDR: %s %v", status, err)
	}
	if seenUser != ruc+"FACT01" || seenPassword != "clave-secreta-123" || !strings.Contains(seenXML, "<ds:Signature") || !strings.Contains(seenXML, ruc) {
		t.Fatalf("send used decrypted credentials and a signed XML: %s %s", seenUser, seenPassword)
	}
	xmlFile := call(e.File, "GET", "", map[string]string{"id": paid.FiscalDocument.ID, "kind": "xml"}, 200)
	if !strings.Contains(xmlFile.Body.String(), "<ds:SignatureValue>") || xmlFile.Header().Get("Content-Disposition") == "" {
		t.Fatal("signed XML download")
	}
	call(e.File, "GET", "", map[string]string{"id": paid.FiscalDocument.ID, "kind": "cdr"}, 200)
	detail := call(e.Get, "GET", "", map[string]string{"id": paid.FiscalDocument.ID}, 200).Body.String()
	if !strings.Contains(detail, `"files":["cdr","xml"]`) || !strings.Contains(detail, `"digestValue":"`) {
		t.Fatalf("detail lists the stored files: %s", detail)
	}
}
