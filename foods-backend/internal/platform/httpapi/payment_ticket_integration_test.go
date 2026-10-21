package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestPaymentTicketIdentityIsScopedAndReadOnly(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	if _, err := pool.Exec(ctx, `UPDATE organizations SET trade_name='La Panchita',legal_name='Panchita Restaurante SAC' WHERE id=$1`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE locations SET address='Av. Perú 123',phone='999888777',timezone='America/Lima' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	var orderID string
	if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by) VALUES($1,$2,'PED-TICKET','mostrador','listo',20,$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	get := func(target scope) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/v1/admin/pos/orders/"+orderID, nil)
		req.SetPathValue("id", orderID)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, target))
		rec := httptest.NewRecorder()
		api.getPOSOrder(rec, req)
		return rec
	}
	rec := get(s)
	if rec.Code != 200 {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body.String())
	}
	var detail posOrderDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	identity := detail.ReceiptContext
	if identity.OrganizationName != "La Panchita" || identity.LegalName != "Panchita Restaurante SAC" || identity.Address != "Av. Perú 123" || identity.Phone != "999888777" || identity.Country != "PE" || identity.Currency != "PEN" || identity.CurrencySymbol == "" || identity.Timezone != "America/Lima" || identity.TaxID == "" {
		t.Fatalf("wrong receipt identity: %#v", identity)
	}
	for _, target := range []scope{other, {OrganizationID: s.OrganizationID, LocationID: other.LocationID, UserID: s.UserID}} {
		if res := get(target); res.Code != 404 {
			t.Fatalf("cross scope: %d %s", res.Code, res.Body.String())
		}
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE order_id=$1`, orderID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("ticket read created payments: %d", count)
	}
	if detail.PaymentStatus != "pending" || len(detail.Payments) != 0 {
		t.Fatalf("invented payment: %#v", detail)
	}
}
