package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// orders.paid_amount (migración 000110) es el cobrado neto que mantienen los
// triggers de cobros y devoluciones; el listado de POS filtra y totaliza con él.
// Esta prueba recorre cobro, devolución y borrado por SQL y comprueba que la
// columna coincide siempre con los documentos y que POS clasifica igual.
func TestOrderPaidAmountFollowsPaymentsAndRefunds(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	foreign := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	grantCashManagementForPaymentTest(t, api, s)

	call := func(handler func(http.ResponseWriter, *http.Request), method, target string, body []byte, pathID string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, target, bytes.NewReader(body))
		if pathID != "" {
			req.SetPathValue("id", pathID)
		}
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	var registerID string
	if err := pool.QueryRow(ctx, `INSERT INTO cash_registers(organization_id,location_id,name,created_by) VALUES($1,$2,'Caja cobrado',$3) RETURNING id`,
		s.OrganizationID, s.LocationID, s.UserID).Scan(&registerID); err != nil {
		t.Fatal(err)
	}
	if rec := call(api.openCashShift, "POST", "/v1/admin/cash-shifts", []byte(fmt.Sprintf(`{"cashRegisterId":%q,"openingAmount":0}`, registerID)), ""); rec.Code != 201 {
		t.Fatalf("open shift: %d %s", rec.Code, rec.Body.String())
	}
	order := func(total string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by) VALUES($1,$2,$3,'mostrador','listo',$4,$5) RETURNING id`,
			s.OrganizationID, s.LocationID, fmt.Sprintf("PED-PAID-%d", time.Now().UnixNano()), total, s.UserID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	// paid devuelve la columna y el cálculo desde los documentos, como texto.
	paid := func(orderID string) (string, string) {
		t.Helper()
		var column, documents string
		if err := pool.QueryRow(ctx, `
			SELECT o.paid_amount::text, COALESCE((
			  SELECT sum(p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id),0))
			  FROM payments p WHERE p.order_id=o.id AND p.location_id=o.location_id),0)::text
			FROM orders o WHERE o.id=$1`, orderID).Scan(&column, &documents); err != nil {
			t.Fatal(err)
		}
		return column, documents
	}
	expectPaid := func(orderID, want string) {
		t.Helper()
		if column, documents := paid(orderID); column != want || documents != want {
			t.Fatalf("paid_amount=%s documents=%s, want %s", column, documents, want)
		}
	}
	type posList struct {
		Items []struct {
			ID              string   `json:"id"`
			PaidAmount      string   `json:"paidAmount"`
			RemainingAmount string   `json:"remainingAmount"`
			PaymentStatus   string   `json:"paymentStatus"`
			PaymentMethods  []string `json:"paymentMethods"`
		} `json:"items"`
		Total       int    `json:"total"`
		AmountTotal string `json:"amountTotal"`
	}
	list := func(status string) posList {
		t.Helper()
		rec := call(api.listPOSOrders, "GET", "/v1/admin/pos/orders?pageSize=50&paymentStatus="+status, nil, "")
		if rec.Code != 200 {
			t.Fatalf("pos list %s: %d %s", status, rec.Code, rec.Body.String())
		}
		var out posList
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	unpaidOrder := order("30")
	paidOrder := order("60")
	expectPaid(paidOrder, "0")

	pay := func(orderID, method string, amount int) string {
		t.Helper()
		rec := call(api.createPayment, "POST", "/v1/admin/payments", []byte(fmt.Sprintf(`{"orderId":%q,"method":%q,"amount":%d}`, orderID, method, amount)), "")
		if rec.Code != 201 {
			t.Fatalf("payment: %d %s", rec.Code, rec.Body.String())
		}
		var payment paymentView
		if err := json.Unmarshal(rec.Body.Bytes(), &payment); err != nil {
			t.Fatal(err)
		}
		return payment.ID
	}
	cash := pay(paidOrder, "cash", 40)
	expectPaid(paidOrder, "40.00")
	pay(paidOrder, "card", 20)
	expectPaid(paidOrder, "60.00")

	unpaid := list("unpaid")
	if unpaid.Total != 1 || len(unpaid.Items) != 1 || unpaid.Items[0].ID != unpaidOrder || unpaid.Items[0].PaidAmount != "0" ||
		unpaid.Items[0].RemainingAmount != "30.00" || unpaid.Items[0].PaymentStatus != "pending" || len(unpaid.Items[0].PaymentMethods) != 0 || unpaid.AmountTotal != "0" {
		t.Fatalf("unpaid list: %+v", unpaid)
	}
	if done := list("paid"); done.Total != 1 || done.Items[0].ID != paidOrder || done.Items[0].PaidAmount != "60.00" || len(done.Items[0].PaymentMethods) != 2 || done.AmountTotal != "60.00" {
		t.Fatalf("paid list: %+v", done)
	}

	if rec := call(api.refundPayment, "POST", "/v1/admin/payments/"+cash+"/refund", []byte(`{"amount":40,"reason":"Corrección de cobro"}`), cash); rec.Code != 204 {
		t.Fatalf("refund: %d %s", rec.Code, rec.Body.String())
	}
	expectPaid(paidOrder, "20.00")
	partial := list("partial")
	if partial.Total != 1 || partial.Items[0].ID != paidOrder || partial.Items[0].RemainingAmount != "40.00" || partial.Items[0].PaymentStatus != "partial" {
		t.Fatalf("partial list: %+v", partial)
	}
	// El medio totalmente devuelto deja de mostrarse; el saldo vuelve a «por cobrar».
	if len(partial.Items[0].PaymentMethods) != 1 {
		t.Fatalf("fully refunded method still listed: %+v", partial.Items[0].PaymentMethods)
	}
	if all := list("unpaid"); all.Total != 2 || all.AmountTotal != "20.00" {
		t.Fatalf("unpaid after refund: %+v", all)
	}

	// Corrección directa en base: el trigger también sigue cambios y borrados.
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE payment_refunds SET amount=30 WHERE payment_id=$1`, cash)
	expectPaid(paidOrder, "30.00")
	exec(`DELETE FROM cash_movements WHERE organization_id=$1 AND source_type='cash_refund'`, s.OrganizationID)
	exec(`DELETE FROM payment_refunds WHERE payment_id=$1`, cash)
	expectPaid(paidOrder, "60.00")
	exec(`DELETE FROM cash_movements WHERE organization_id=$1`, s.OrganizationID)
	exec(`DELETE FROM payments WHERE order_id=$1`, paidOrder)
	expectPaid(paidOrder, "0")

	// Otra empresa no ve ni altera estos pedidos.
	var foreignVisible int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE organization_id=$1 AND paid_amount<>0`, foreign.OrganizationID).Scan(&foreignVisible); err != nil || foreignVisible != 0 {
		t.Fatalf("foreign orders touched: %d %v", foreignVisible, err)
	}
}
