package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A package of 0.001 bottles stored 2 packages as 0.002 bottles and a cost of
// 10 000 per bottle: countable units must reject presentations under one piece.
func TestCountableUnitsRejectFractionalPresentations(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	call := func(method, path, body, id string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}

	units := call("GET", "/v1/admin/purchase-units", "", "", api.listInventoryUnits)
	var catalog struct {
		Items []struct {
			Code      string `json:"code"`
			Countable bool   `json:"countable"`
		} `json:"items"`
	}
	if units.Code != 200 || json.Unmarshal(units.Body.Bytes(), &catalog) != nil {
		t.Fatalf("units: %d %s", units.Code, units.Body.String())
	}
	countable := map[string]bool{}
	for _, unit := range catalog.Items {
		countable[unit.Code] = unit.Countable
	}
	if !countable["botella"] || !countable["und"] || !countable["lata"] || countable["kg"] || countable["ml"] {
		t.Fatalf("unexpected countable catalog: %v", countable)
	}
	combinations := call("GET", "/v1/admin/purchase-combinations?unit=botella", "", "", api.listInventoryCombinations)
	if combinations.Code != 200 || !strings.Contains(combinations.Body.String(), `"unitCountable":true`) {
		t.Fatalf("combinations must expose unitCountable: %d %s", combinations.Code, combinations.Body.String())
	}

	rejected := call("POST", "/v1/admin/purchase-inventory-items", `{"unit":"botella","newIngredient":{"name":"Agua fraccionada"},"presentations":[{"presentationType":"package","unitsPerPresentation":0.001,"isDefault":true}]}`, "", api.createPurchaseInventoryItem)
	if rejected.Code != 400 || !strings.Contains(rejected.Body.String(), "invalid_presentation_content") || !strings.Contains(rejected.Body.String(), "al menos 1 botella") {
		t.Fatalf("fractional bottle package accepted: %d %s", rejected.Code, rejected.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_items WHERE organization_id=$1 AND name='Agua fraccionada'`, s.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rejected presentation left a partial article: %d %v", count, err)
	}

	created := call("POST", "/v1/admin/purchase-inventory-items", `{"unit":"botella","newIngredient":{"name":"Agua por paquete"},"presentations":[{"presentationType":"package","unitsPerPresentation":6,"isDefault":true}]}`, "", api.createPurchaseInventoryItem)
	var item struct {
		ID string `json:"id"`
	}
	if created.Code != 201 || json.Unmarshal(created.Body.Bytes(), &item) != nil || item.ID == "" {
		t.Fatalf("pack of six bottles: %d %s", created.Code, created.Body.String())
	}
	if rec := call("POST", "/v1/admin/purchase-inventory-items/"+item.ID+"/presentations", `{"presentationType":"package","unitsPerPresentation":0.5}`, item.ID, api.savePurchasePresentation); rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_presentation_content") {
		t.Fatalf("adding a fractional presentation later must be rejected: %d %s", rec.Code, rec.Body.String())
	}

	if rec := call("POST", "/v1/admin/purchase-combinations", `{"unit":"lata","code":"sixpack","name":"Six pack"}`, "", api.createInventoryCombination); rec.Code != 201 || !strings.Contains(rec.Body.String(), `"unitCountable":true`) {
		t.Fatalf("a new combination must keep the unit rule: %d %s", rec.Code, rec.Body.String())
	}

	// Measured units keep fractions, e.g. a bag of half a kilo.
	if rec := call("POST", "/v1/admin/purchase-combinations", `{"unit":"kg","code":"package","name":"Paquete"}`, "", api.createInventoryCombination); rec.Code != 201 && rec.Code != 409 {
		t.Fatalf("kg package combination: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("POST", "/v1/admin/purchase-inventory-items", `{"unit":"kg","newIngredient":{"name":"Azúcar media bolsa"},"presentations":[{"presentationType":"package","unitsPerPresentation":0.5,"isDefault":true}]}`, "", api.createPurchaseInventoryItem); rec.Code != 201 {
		t.Fatalf("half-kilo package must be accepted: %d %s", rec.Code, rec.Body.String())
	}
}
