package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPaymentMethodsAreDatabaseDriven(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO payment_methods(
			organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		) VALUES($1,'voucher','Vale interno','Medio creado desde datos',true,true,true,false,55)
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	method, err := getPaymentMethod(ctx, pool, s.OrganizationID, "voucher", "sales")
	if err != nil {
		t.Fatalf("database-defined payment method should be accepted: %v", err)
	}
	if method.Code != "voucher" || method.Name != "Vale interno" || method.AffectsCash {
		t.Fatalf("unexpected method: %#v", method)
	}
	if _, err := pool.Exec(ctx, `UPDATE payment_methods SET active=false WHERE organization_id=$1 AND code='voucher'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := getPaymentMethod(ctx, pool, s.OrganizationID, "voucher", "sales"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("inactive database method must be rejected, got %v", err)
	}
}

func TestPaymentMethodAdminLifecycle(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)

	createReq := httptest.NewRequest("POST", "/v1/admin/payment-methods", bytes.NewBufferString(`{
		"code":"voucher",
		"name":"Vale interno",
		"description":"Convenio corporativo",
		"salesEnabled":true,
		"expensesEnabled":false,
		"affectsCash":false,
		"sortOrder":60
	}`))
	createReq = createReq.WithContext(context.WithValue(createReq.Context(), scopeKey{}, s))
	createRec := httptest.NewRecorder()
	api.createPaymentMethod(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create payment method: %d %s", createRec.Code, createRec.Body.String())
	}
	var created paymentMethodView
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Code != "voucher" || created.Name != "Vale interno" || !created.SalesEnabled || created.ExpensesEnabled {
		t.Fatalf("unexpected created method: %#v", created)
	}

	updateReq := httptest.NewRequest("PATCH", "/v1/admin/payment-methods/voucher", bytes.NewBufferString(`{
		"name":"Vale empresa",
		"description":"Convenio actualizado",
		"salesEnabled":true,
		"expensesEnabled":true,
		"affectsCash":false,
		"sortOrder":65
	}`))
	updateReq.SetPathValue("code", "voucher")
	updateReq = updateReq.WithContext(context.WithValue(updateReq.Context(), scopeKey{}, s))
	updateRec := httptest.NewRecorder()
	api.updatePaymentMethod(updateRec, updateReq)
	if updateRec.Code != 200 {
		t.Fatalf("update payment method: %d %s", updateRec.Code, updateRec.Body.String())
	}

	statusReq := httptest.NewRequest("PATCH", "/v1/admin/payment-methods/voucher/status", bytes.NewBufferString(`{"active":false}`))
	statusReq.SetPathValue("code", "voucher")
	statusReq = statusReq.WithContext(context.WithValue(statusReq.Context(), scopeKey{}, s))
	statusRec := httptest.NewRecorder()
	api.updatePaymentMethodStatus(statusRec, statusReq)
	if statusRec.Code != 204 {
		t.Fatalf("deactivate payment method: %d %s", statusRec.Code, statusRec.Body.String())
	}
	if _, err := getPaymentMethod(context.Background(), pool, s.OrganizationID, "voucher", "sales"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("deactivated method must not be operational, got %v", err)
	}
}

func TestPaymentMethodAdminProtectsLastOperationalMethod(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		UPDATE payment_methods
		SET active=(code='cash'),sales_enabled=(code='cash'),expenses_enabled=(code='cash')
		WHERE organization_id=$1
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PATCH", "/v1/admin/payment-methods/cash/status", bytes.NewBufferString(`{"active":false}`))
	req.SetPathValue("code", "cash")
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updatePaymentMethodStatus(rec, req)
	if rec.Code != 409 {
		t.Fatalf("expected continuity conflict, got %d %s", rec.Code, rec.Body.String())
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT active FROM payment_methods WHERE organization_id=$1 AND code='cash'`, s.OrganizationID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("rollback must keep the last operational method active")
	}
}
