package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCashClosingReportDetailsSnapshotAndIsolation(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	call := func(method, path, body, id string, target scope, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, target))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	register := call("POST", "/", `{"name":"Caja reporte"}`, "", s, api.createCashRegister)
	var cr cashRegisterView
	_ = json.Unmarshal(register.Body.Bytes(), &cr)
	opened := call("POST", "/", fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":100}`, cr.ID), "", s, api.openCashShift)
	if opened.Code != 201 {
		t.Fatalf("open: %d %s", opened.Code, opened.Body.String())
	}
	var shift cashShiftView
	_ = json.Unmarshal(opened.Body.Bytes(), &shift)
	var orderID string
	if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,customer_name,subtotal,total,created_by) VALUES($1,$2,'mostrador','entregado','Cliente reporte',60,60,$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,sku,name,price) VALUES($1,'REPORT-PRODUCT','Ají de gallina',30) RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO order_items(organization_id,order_id,product_id,name,qty,unit_price,note) VALUES($1,$2,$3,'Ají de gallina',2,30,'Sin ají')`, s.OrganizationID, orderID, productID); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{fmt.Sprintf(`{"orderId":%q,"method":"cash","amount":30}`, orderID), fmt.Sprintf(`{"orderId":%q,"method":"card","amount":20}`, orderID)} {
		rec := call("POST", "/", body, "", s, api.createPayment)
		if rec.Code != 201 {
			t.Fatalf("payment: %d %s", rec.Code, rec.Body.String())
		}
	}
	rec := call("POST", "/", `{"movementType":"expense","amount":10,"reason":"Compra de hielo","note":"Una bolsa"}`, shift.ID, s, api.createCashMovement)
	if rec.Code != 201 {
		t.Fatalf("expense: %s", rec.Body.String())
	}
	rec = call("GET", "/", "", shift.ID, s, api.getCashShiftReport)
	if rec.Code != 200 {
		t.Fatalf("report: %d %s", rec.Code, rec.Body.String())
	}
	var report cashShiftReport
	_ = json.Unmarshal(rec.Body.Bytes(), &report)
	if report.Persisted || len(report.Sales) != 1 || report.Sales[0].Quantity != "2.00" || report.Sales[0].Total != "60.00" || len(report.Payments) != 2 || report.CollectedAmount != "50.00" || report.Shift.ExpectedAmount != "120.00" || report.Shift.ExpenseAmount != "10.00" {
		t.Fatalf("incorrect partial/split report: %+v", report)
	}
	if len(report.Methods) != 2 || len(report.Shift.Movements) != 2 || report.Shift.OpenedByName == "" || report.Timezone == "" {
		t.Fatal("missing details")
	}
	if _, err := time.Parse(time.RFC3339Nano, report.Sales[0].CreatedAt); err != nil {
		t.Fatal(err)
	}
	other := seedInventoryScope(t, pool)
	if response := call("GET", "/", "", shift.ID, other, api.getCashShiftReport); response.Code != 404 {
		t.Fatal("report crossed tenant")
	}
	local := s
	local.LocationID = other.LocationID
	if response := call("GET", "/", "", shift.ID, local, api.getCashShiftReport); response.Code != 404 {
		t.Fatal("report crossed local")
	}
	// Simulate persistence failure: neither closure, counts nor unassignment survive.
	constraint := "cash_report_reject_" + strings.ReplaceAll(s.OrganizationID, "-", "")
	if _, err := pool.Exec(ctx, fmt.Sprintf(`ALTER TABLE cash_shift_reports ADD CONSTRAINT %s CHECK (organization_id <> '%s'::uuid) NOT VALID`, constraint, s.OrganizationID)); err != nil {
		t.Fatal(err)
	}
	failed := call("POST", "/", `{"counts":[{"denomination":20,"quantity":6}],"note":"Conteo"}`, shift.ID, s, api.closeCashShift)
	_, _ = pool.Exec(ctx, `ALTER TABLE cash_shift_reports DROP CONSTRAINT `+constraint)
	if failed.Code != 503 {
		t.Fatalf("rollback: %d %s", failed.Code, failed.Body.String())
	}
	var status string
	var counts int
	_ = pool.QueryRow(ctx, `SELECT status FROM cash_shifts WHERE id=$1`, shift.ID).Scan(&status)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM cash_count_lines WHERE shift_id=$1`, shift.ID).Scan(&counts)
	if status != "open" || counts != 0 {
		t.Fatal("report error did not roll back closing")
	}
	closed := call("POST", "/", `{"counts":[{"denomination":20,"quantity":6}],"note":"Conteo confirmado"}`, shift.ID, s, api.closeCashShift)
	if closed.Code != 200 {
		t.Fatalf("close: %d %s", closed.Code, closed.Body.String())
	}
	rec = call("GET", "/", "", shift.ID, s, api.getCashShiftReport)
	_ = json.Unmarshal(rec.Body.Bytes(), &report)
	if rec.Code != 200 || !report.Persisted || report.Shift.Status != "closed" || len(report.Counts) != 1 || report.Shift.ClosingCountedAmount == nil || *report.Shift.ClosingCountedAmount != "120.00" {
		t.Fatalf("snapshot: %+v", report)
	}
	before := rec.Body.String()
	if _, err := pool.Exec(ctx, `UPDATE users SET full_name='Nombre modificado' WHERE id=$1`, s.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE payment_methods SET name='Medio modificado' WHERE organization_id=$1`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if after := call("GET", "/", "", shift.ID, s, api.getCashShiftReport); after.Body.String() != before {
		t.Fatal("issued report changed after catalog rename")
	}
	if again := call("POST", "/", `{"countedAmount":120}`, shift.ID, s, api.closeCashShift); again.Code != 409 {
		t.Fatal("closed shift accepted duplicate")
	}
}

func TestCashReportReadPermissionAndBlindClosing(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	var register, id, role string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,blind_close,created_by) VALUES($1,$2,'Caja ciega reporte',true,$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&register); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO cash_shifts(organization_id,location_id,cash_register_id,opened_by,business_date) VALUES($1,$2,$3,$4,current_date) RETURNING id`, s.OrganizationID, s.LocationID, register, s.UserID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Cash report reader',ARRAY['cash.read']) RETURNING id`, s.OrganizationID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, role, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "cash-report-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := api.Routes()
	request := func(auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/v1/admin/cash-shifts/"+id+"/report", nil)
		if auth {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
	if rec := request(false); rec.Code != 401 {
		t.Fatal("anonymous report accessible")
	}
	if rec := request(true); rec.Code != 403 || !strings.Contains(rec.Body.String(), "cash_report_blind_close") {
		t.Fatalf("blind report leaked: %d %s", rec.Code, rec.Body.String())
	}
	_, _ = pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['cash.read','cash.expected.read'] WHERE id=$1`, role)
	if rec := request(true); rec.Code != 200 {
		t.Fatalf("supervisor report: %s", rec.Body.String())
	}
	_, _ = pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY[]::text[] WHERE id=$1`, role)
	if rec := request(true); rec.Code != 403 {
		t.Fatal("missing read accepted")
	}
	_, _ = pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['cash.read'] WHERE id=$1`, role)
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"countedAmount":0}`))
	req.SetPathValue("id", id)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.closeCashShift(rec, req)
	if rec.Code != 200 {
		t.Fatalf("blind close: %d %s", rec.Code, rec.Body.String())
	}
	if rec = request(true); rec.Code != 200 {
		t.Fatalf("closed blind report: %s", rec.Body.String())
	}
}
