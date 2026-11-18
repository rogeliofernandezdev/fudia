package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRecipeConsumptionUnitsPersistConvertConsumeAndReverse(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	call := func(actor scope, method, path, id, body string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.SetPathValue("id", id)
		req.SetPathValue("productId", id)
		req = req.WithContext(context.WithValue(ctx, scopeKey{}, actor))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	if _, err := pool.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'inventario',true),($1,'recetas',true) ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, item := range []struct{ name, unit string }{{"Carne", "kg"}, {"Aceite", "l"}, {"Especia", "kg"}} {
		rec := call(s, "POST", "/v1/admin/purchase-inventory-items", "", fmt.Sprintf(`{"newIngredient":{"name":%q},"unit":%q}`, item.name, item.unit), api.createPurchaseInventoryItem)
		var result inventoryProductOption
		if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &result) != nil {
			t.Fatalf("seed item: %d %s", rec.Code, rec.Body.String())
		}
		ids[item.name] = result.ID
		if _, err := pool.Exec(ctx, `UPDATE stock_balances SET quantity=1,average_unit_cost=100 WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, result.ID); err != nil {
			t.Fatal(err)
		}
	}
	var productID string
	if err := pool.QueryRow(ctx, `INSERT INTO products(organization_id,name,sku,price,quantity_control) VALUES($1,'Plato con conversiones','recipe-units-test',20,'none') RETURNING id`, s.OrganizationID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"yieldQuantity":2,"active":true,"notes":"Prueba de unidades","items":[
 {"inventoryItemId":%q,"consumptionQuantity":200,"consumptionUnit":"g","quantity":999,"wastePercent":10},
 {"inventoryItemId":%q,"consumptionQuantity":40,"consumptionUnit":"ml"},
 {"inventoryItemId":%q,"consumptionQuantity":0.5,"consumptionUnit":"g"}]}`, ids["Carne"], ids["Aceite"], ids["Especia"])
	saved := call(s, "PUT", "/v1/admin/recipes/"+productID, productID, body, api.saveRecipe)
	if saved.Code != 200 {
		t.Fatalf("save: %d %s", saved.Code, saved.Body.String())
	}
	read := call(s, "GET", "/v1/admin/recipes/"+productID, productID, "", api.getRecipe)
	var recipe struct {
		Items []struct{ InventoryItemID, Quantity, Unit, ConsumptionUnit, ConsumptionQuantity, ConsumptionFactor string }
	}
	if read.Code != 200 || json.Unmarshal(read.Body.Bytes(), &recipe) != nil {
		t.Fatalf("read: %d %s", read.Code, read.Body.String())
	}
	expected := map[string]struct{ base, consumed, unit string }{ids["Carne"]: {"0.200000", "200.000000", "g"}, ids["Aceite"]: {"0.040000", "40.000000", "ml"}, ids["Especia"]: {"0.000500", "0.500000", "g"}}
	for _, item := range recipe.Items {
		want := expected[item.InventoryItemID]
		if item.Quantity != want.base || item.ConsumptionQuantity != want.consumed || item.ConsumptionUnit != want.unit || item.ConsumptionFactor != "0.001000000" {
			t.Fatalf("round trip: %#v", item)
		}
	}
	checkBalance := func(name string, want float64) {
		t.Helper()
		var balance float64
		if err := pool.QueryRow(ctx, `SELECT quantity::float8 FROM stock_balances WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, ids[name]).Scan(&balance); err != nil || math.Abs(balance-want) > 1e-9 {
			t.Fatalf("%s balance: %v want %v err=%v", name, balance, want, err)
		}
	}
	for name := range ids {
		checkBalance(name, 1)
	}
	// Invalid or foreign ingredients/units cannot replace a valid recipe.
	for _, invalid := range []string{
		fmt.Sprintf(`{"inventoryItemId":%q,"consumptionQuantity":1,"consumptionUnit":"l"}`, ids["Carne"]),
		fmt.Sprintf(`{"inventoryItemId":%q,"consumptionQuantity":0.0001,"consumptionUnit":"g"}`, ids["Carne"]),
		fmt.Sprintf(`{"inventoryItemId":%q,"consumptionUnit":"g"}`, ids["Carne"]),
		fmt.Sprintf(`{"inventoryItemId":%q,"consumptionQuantity":1,"consumptionUnit":"missing"}`, ids["Carne"]),
	} {
		rec := call(s, "PUT", "/v1/admin/recipes/"+productID, productID, `{"yieldQuantity":1,"active":true,"items":[`+invalid+`]}`, api.saveRecipe)
		if rec.Code != 400 {
			t.Fatalf("invalid conversion: %d %s", rec.Code, rec.Body.String())
		}
	}
	foreign := call(other, "PUT", "/v1/admin/recipes/"+productID, productID, body, api.saveRecipe)
	if foreign.Code != 404 {
		t.Fatalf("foreign recipe: %d %s", foreign.Code, foreign.Body.String())
	}
	// The same conversion catalog is available without granting Purchasing permissions.
	units := call(s, "GET", "/v1/admin/inventory/units", "", "", api.listInventoryUnits)
	if units.Code != 200 || !strings.Contains(units.Body.String(), `"family":"mass"`) || !strings.Contains(units.Body.String(), `"recipeDefault":true`) {
		t.Fatalf("unit catalog: %d %s", units.Code, units.Body.String())
	}
	orderResponse := call(s, "POST", "/v1/admin/orders", "", fmt.Sprintf(`{"channel":"mostrador","items":[{"productId":%q,"qty":2,"selections":[]}]}`, productID), api.createOrder)
	var order struct{ ID string }
	if orderResponse.Code != 201 || json.Unmarshal(orderResponse.Body.Bytes(), &order) != nil {
		t.Fatalf("order: %d %s", orderResponse.Code, orderResponse.Body.String())
	}
	checkBalance("Carne", 0.78)
	checkBalance("Aceite", 0.96)
	checkBalance("Especia", 0.9995)
	var delta, cost float64
	if err := pool.QueryRow(ctx, `SELECT quantity_delta::float8,value_delta::float8 FROM stock_movements WHERE source_id=$1 AND inventory_item_id=$2 AND movement_type='recipe_consumption'`, order.ID, ids["Especia"]).Scan(&delta, &cost); err != nil || delta != -0.0005 || cost != -0.05 {
		t.Fatalf("small consumption/cost: %v %v %v", delta, cost, err)
	}
	// Later adjustments preserve the remainder, including their document snapshots.
	adjusted := call(s, "POST", "/v1/admin/inventory/adjustments", "", fmt.Sprintf(`{"inventoryItemId":%q,"movementType":"entry","reason":"surplus_adjustment","quantity":0.001,"unitCost":100}`, ids["Especia"]), api.createInventoryAdjustment)
	if adjusted.Code != 201 {
		t.Fatalf("adjustment: %d %s", adjusted.Code, adjusted.Body.String())
	}
	checkBalance("Especia", 1.0005)
	var before, after float64
	if err := pool.QueryRow(ctx, `SELECT stock_before::float8,stock_after::float8 FROM inventory_adjustments WHERE inventory_item_id=$1`, ids["Especia"]).Scan(&before, &after); err != nil || before != 0.9995 || after != 1.0005 {
		t.Fatalf("adjustment snapshots: %v %v %v", before, after, err)
	}
	cancelled := call(s, "PATCH", "/v1/admin/orders/"+order.ID+"/status", order.ID, `{"status":"cancelado"}`, api.updateOrderStatus)
	if cancelled.Code != 200 {
		t.Fatalf("cancel: %d %s", cancelled.Code, cancelled.Body.String())
	}
	checkBalance("Carne", 1)
	checkBalance("Aceite", 1)
	checkBalance("Especia", 1.001)
	// A destructive precision downgrade must refuse to discard fractional stock/history.
	down, err := os.ReadFile("../../../migrations/000134_recipe_consumption_units.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, string(down)); err == nil {
		t.Fatal("precision downgrade unexpectedly succeeded")
	}
	_ = tx.Rollback(ctx)
	// Legacy clients continue to send base units and receive explicit consumption metadata.
	legacy := call(s, "PUT", "/v1/admin/recipes/"+productID, productID, fmt.Sprintf(`{"yieldQuantity":1,"active":true,"items":[{"inventoryItemId":%q,"quantity":0.02}]}`, ids["Carne"]), api.saveRecipe)
	if legacy.Code != 200 {
		t.Fatalf("legacy save: %d %s", legacy.Code, legacy.Body.String())
	}
	legacyRead := call(s, "GET", "/v1/admin/recipes/"+productID, productID, "", api.getRecipe)
	if !strings.Contains(legacyRead.Body.String(), `"consumptionUnit":"kg"`) || !strings.Contains(legacyRead.Body.String(), `"quantity":"0.020000"`) {
		t.Fatalf("legacy read: %s", legacyRead.Body.String())
	}
}
