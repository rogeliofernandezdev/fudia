package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func activeMethodCodes(t *testing.T, s scopeForInventory) []string {
	t.Helper()
	rows, err := s.pool.Query(context.Background(), `SELECT code FROM payment_methods WHERE organization_id=$1 AND active ORDER BY sort_order,code`, s.OrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	codes := []string{}
	for rows.Next() {
		var code string
		if err = rows.Scan(&code); err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
	}
	return codes
}

func TestPaymentMethodsSeedByCountryAndLimitActiveToFour(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	api := New(pool)

	// Companies default to Peru: cash, card, Yape and Plin.
	peru := scopeForInventory{seedInventoryScope(t, pool), pool}
	got := activeMethodCodes(t, peru)
	if want := []string{"cash", "card", "yape", "plin"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		t.Fatalf("Peru must seed %v, got %v", want, got)
	}

	// Checkout reads the same catalog through the admin route.
	req := httptest.NewRequest("GET", "/v1/admin/payment-methods/operational?usage=sales", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, peru.scope))
	rec := httptest.NewRecorder()
	api.listOperationalPaymentMethods(rec, req)
	var list struct {
		Items []struct {
			Code        string `json:"code"`
			Name        string `json:"name"`
			AffectsCash bool   `json:"affectsCash"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != 200 || len(list.Items) != 4 || !list.Items[0].AffectsCash || list.Items[2].Name != "Yape" {
		t.Fatalf("operational catalog: %d %s %v", rec.Code, rec.Body.String(), err)
	}

	// Other countries start with Efectivo and Tarjeta only.
	general := scopeForInventory{seedInventoryScope(t, pool), pool}
	if _, err := pool.Exec(ctx, `UPDATE organization_fiscal_profiles SET country_code='CO' WHERE organization_id=$1`, general.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM payment_methods WHERE organization_id=$1`, general.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err := seedOrganizationPaymentMethods(ctx, pool, general.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if codes := activeMethodCodes(t, general); len(codes) != 2 || codes[0] != "cash" || codes[1] != "card" {
		t.Fatalf("general set must contain cash and card only: %v", codes)
	}
	var seededTotal int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payment_methods WHERE organization_id=$1`, general.OrganizationID).Scan(&seededTotal); err != nil || seededTotal != 2 {
		t.Fatalf("general catalog must contain two entries in total: %d %v", seededTotal, err)
	}

	// A fifth active method is rejected.
	if _, err := pool.Exec(ctx, `
		INSERT INTO payment_methods(organization_id,code,name,active,sales_enabled,expenses_enabled,affects_cash,sort_order)
		VALUES($1,'transfer','Transferencia',false,true,true,false,50)`, peru.OrganizationID); err != nil {
		t.Fatal(err)
	}
	statusReq := httptest.NewRequest("PATCH", "/v1/admin/payment-methods/transfer/status", bytes.NewReader([]byte(`{"active":true}`)))
	statusReq.SetPathValue("code", "transfer")
	statusReq = statusReq.WithContext(context.WithValue(statusReq.Context(), scopeKey{}, peru.scope))
	statusRec := httptest.NewRecorder()
	api.updatePaymentMethodStatus(statusRec, statusReq)
	if statusRec.Code != 409 || !bytes.Contains(statusRec.Body.Bytes(), []byte("payment_method_limit")) {
		t.Fatalf("a fifth active method must be rejected: %d %s", statusRec.Code, statusRec.Body.String())
	}
}
