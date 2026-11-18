package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPurchaseItemSavesStockLevelsForActiveLocation(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	category := seedRetailCategory(t, pool, s)
	call := func(actor scope, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/purchase-inventory-items", strings.NewReader(body))
		req = req.WithContext(context.WithValue(ctx, scopeKey{}, actor))
		rec := httptest.NewRecorder()
		api.createPurchaseInventoryItem(rec, req)
		return rec
	}
	for _, tc := range []struct {
		name, source, unit, levels string
		minimum, optimal           float64
	}{
		{"ingredient", `"newIngredient":{"name":"Tomate niveles"}`, "kg", `"minimumStock":5,"optimalStock":15.5`, 5, 15.5},
		{"retail", fmt.Sprintf(`"newProduct":{"name":"Agua niveles","categoryId":%q,"price":"5"}`, category), "botella", `"minimumStock":5,"optimalStock":15`, 5, 15},
		{"legacy minimum", `"newIngredient":{"name":"Arroz sin óptimo"}`, "kg", `"minimumStock":2`, 2, 2},
		{"normalized", `"newIngredient":{"name":"Azúcar niveles"}`, "kg", `"minimumStock":5,"optimalStock":2`, 5, 5},
		{"zero", `"newIngredient":{"name":"Sal niveles"}`, "kg", `"optimalStock":0`, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := call(s, fmt.Sprintf(`{%s,"unit":%q,%s}`, tc.source, tc.unit, tc.levels))
			var item inventoryProductOption
			if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &item) != nil {
				t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
			}
			var minimum, optimal, quantity float64
			var movements, settings int
			if err := pool.QueryRow(ctx, `SELECT l.minimum_stock::float8,l.optimal_stock::float8,b.quantity::float8,
			 (SELECT count(*) FROM stock_movements WHERE inventory_item_id=$2),
			 (SELECT count(*) FROM inventory_location_settings WHERE inventory_item_id=$2)
			 FROM inventory_location_settings l JOIN stock_balances b USING(organization_id,location_id,inventory_item_id)
			 WHERE l.organization_id=$1 AND l.inventory_item_id=$2 AND l.location_id=$3`, s.OrganizationID, item.ID, s.LocationID).Scan(&minimum, &optimal, &quantity, &movements, &settings); err != nil {
				t.Fatal(err)
			}
			if minimum != tc.minimum || optimal != tc.optimal || quantity != 0 || movements != 0 || settings != 1 {
				t.Fatalf("levels/balance: min=%v optimal=%v quantity=%v movements=%d settings=%d", minimum, optimal, quantity, movements, settings)
			}
			var response map[string]any
			_ = json.Unmarshal(rec.Body.Bytes(), &response)
			if fmt.Sprint(response["optimalStock"]) != fmt.Sprint(tc.optimal) {
				t.Fatalf("response optimal: %s", rec.Body.String())
			}
			// The same Inventory editor changes these levels, scoped to another local.
			var locationID string
			if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code) VALUES($1,$2,$3) RETURNING id`, s.OrganizationID, "Otro local "+tc.name, "LEVEL-"+item.ID).Scan(&locationID); err != nil {
				t.Fatal(err)
			}
			local := s
			local.LocationID = locationID
			req := httptest.NewRequest("PATCH", "/v1/admin/inventory/items/"+item.ID+"/settings", strings.NewReader(`{"minimumStock":1,"optimalStock":3}`))
			req.SetPathValue("id", item.ID)
			req = req.WithContext(context.WithValue(ctx, scopeKey{}, local))
			updated := httptest.NewRecorder()
			api.updateInventoryLocationSettings(updated, req)
			if updated.Code != 204 {
				t.Fatalf("update: %d %s", updated.Code, updated.Body.String())
			}
			if err := pool.QueryRow(ctx, `SELECT optimal_stock::float8 FROM inventory_location_settings WHERE inventory_item_id=$1 AND location_id=$2`, item.ID, s.LocationID).Scan(&optimal); err != nil || optimal != tc.optimal {
				t.Fatalf("another local changed the initial target: %v %v", optimal, err)
			}
			req = httptest.NewRequest("PATCH", "/v1/admin/inventory/items/"+item.ID+"/settings", strings.NewReader(`{"minimumStock":1,"optimalStock":3}`))
			req.SetPathValue("id", item.ID)
			req = req.WithContext(context.WithValue(ctx, scopeKey{}, other))
			foreign := httptest.NewRecorder()
			api.updateInventoryLocationSettings(foreign, req)
			if foreign.Code != 404 {
				t.Fatalf("foreign settings: %d %s", foreign.Code, foreign.Body.String())
			}
		})
	}
	for i, levels := range []string{`"minimumStock":1,"optimalStock":-1`, `"optimalStock":100000000000`, `"minimumStock":0.5,"optimalStock":5`, `"minimumStock":1,"optimalStock":2.5`} {
		name := fmt.Sprintf("Rejected stock levels %d", i)
		rec := call(s, fmt.Sprintf(`{"newIngredient":{"name":%q},"unit":"botella",%s}`, name, levels))
		if rec.Code != 400 {
			t.Fatalf("invalid levels: %d %s", rec.Code, rec.Body.String())
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_items WHERE organization_id=$1 AND name=$2`, s.OrganizationID, name).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rejected article persisted: %d %v", count, err)
		}
	}
}
