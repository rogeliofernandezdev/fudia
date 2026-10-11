package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSalesListPaymentMethods(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	var registerID, orderID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by)
		VALUES($1,$2,'Caja ventas',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	request := func(method, path string, body []byte, actor scope) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, actor))
		w := httptest.NewRecorder()
		switch method {
		case "GET":
			api.listPOSOrders(w, r)
		default:
			api.openCashShift(w, r)
		}
		return w
	}
	opened := request("POST", "/v1/admin/cash-shifts", []byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID)), s)
	if opened.Code != 201 {
		t.Fatalf("open shift: %d %s", opened.Code, opened.Body.String())
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by)
		VALUES($1,$2,'PED-METHODS','mostrador','listo',60,$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	list := func(actor scope, status string) []posOrderSummary {
		t.Helper()
		w := request("GET", "/v1/admin/pos/orders?paymentStatus="+status+"&pageSize=1", nil, actor)
		if w.Code != 200 {
			t.Fatalf("list: %d %s", w.Code, w.Body.String())
		}
		var result struct {
			Items []posOrderSummary `json:"items"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Total != len(result.Items) {
			t.Fatalf("payment joins must not duplicate rows: %s", w.Body.String())
		}
		return result.Items
	}
	assertMethods := func(status string, want []string) {
		t.Helper()
		items := list(s, status)
		if len(items) != 1 || items[0].ID != orderID || !reflect.DeepEqual(items[0].PaymentMethods, want) {
			t.Fatalf("expected methods %v, got %#v", want, items)
		}
	}
	assertMethods("all", []string{})
	pay := func(method string, amount int) string {
		t.Helper()
		r := httptest.NewRequest("POST", "/v1/admin/payments", bytes.NewBufferString(fmt.Sprintf(`{"orderId":%q,"method":%q,"amount":%d}`, orderID, method, amount)))
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		api.createPayment(w, r)
		if w.Code != 201 {
			t.Fatalf("payment: %d %s", w.Code, w.Body.String())
		}
		var payment paymentView
		if err := json.Unmarshal(w.Body.Bytes(), &payment); err != nil {
			t.Fatal(err)
		}
		return payment.ID
	}
	// Use tenant-specific labels, not a frontend/global method dictionary.
	if _, err := pool.Exec(ctx, `UPDATE payment_methods SET name=CASE code WHEN 'cash' THEN 'A efectivo local' ELSE 'B convenio local' END
		WHERE organization_id=$1 AND code IN ('cash','card')`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	first := pay("cash", 10)
	second := pay("cash", 10)
	pay("card", 40)
	assertMethods("paid", []string{"A efectivo local", "B convenio local"})
	if _, err := pool.Exec(ctx, `UPDATE payment_methods SET active=false WHERE organization_id=$1 AND code='card'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	assertMethods("paid", []string{"A efectivo local", "B convenio local"})
	refund := func(id string) {
		t.Helper()
		r := httptest.NewRequest("POST", "/v1/admin/payments/"+id+"/refund", bytes.NewBufferString(`{"amount":10,"reason":"Corrección"}`))
		r.SetPathValue("id", id)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		api.refundPayment(w, r)
		if w.Code != 204 {
			t.Fatalf("refund: %d %s", w.Code, w.Body.String())
		}
	}
	refund(first)
	assertMethods("partial", []string{"A efectivo local", "B convenio local"})
	refund(second)
	assertMethods("partial", []string{"B convenio local"})
	if len(list(s, "paid")) != 0 {
		t.Fatal("refunded sale must not remain in paid history")
	}
	other := seedInventoryScope(t, pool)
	if len(list(other, "all")) != 0 {
		t.Fatal("another organization must not see sale or payment methods")
	}
	otherLocal := s
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address)
		VALUES($1,'Otro local','OTHER','') RETURNING id`, s.OrganizationID).Scan(&otherLocal.LocationID); err != nil {
		t.Fatal(err)
	}
	if len(list(otherLocal, "all")) != 0 {
		t.Fatal("another location must not see sale or payment methods")
	}
}
