package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInventorySettingsRespectUnitCountability(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	for _, unit := range []string{"botella", "kg"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active) VALUES($1,$2,$2,$3,0,true) RETURNING id`, s.OrganizationID, "Niveles "+unit, unit).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for _, body := range []string{`{"minimumStock":0.5,"reorderPoint":2,"optimalStock":3}`, `{"minimumStock":1,"reorderPoint":0.5,"optimalStock":3}`, `{"minimumStock":1,"reorderPoint":2,"optimalStock":0.5}`} {
			req := httptest.NewRequest("PATCH", "/v1/admin/inventory/items/"+id+"/settings", strings.NewReader(body))
			req.SetPathValue("id", id)
			req = req.WithContext(context.WithValue(ctx, scopeKey{}, s))
			rec := httptest.NewRecorder()
			api.updateInventoryLocationSettings(rec, req)
			if unit == "botella" {
				if rec.Code != 400 || !strings.Contains(rec.Body.String(), "fractional_inventory_settings") {
					t.Fatalf("bottle fractions: %d %s", rec.Code, rec.Body.String())
				}
			} else if rec.Code != 204 {
				t.Fatalf("measured fractions: %d %s", rec.Code, rec.Body.String())
			}
		}
		if unit == "botella" {
			var count int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_location_settings WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, id).Scan(&count); err != nil || count != 0 {
				t.Fatalf("rejected settings persisted: count=%d err=%v", count, err)
			}
			// New clients omit the retired field. A legacy high reorder point must
			// not force the user to keep an unnecessarily high optimal level.
			if _, err := pool.Exec(ctx, `INSERT INTO inventory_location_settings(organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock) VALUES($1,$2,$3,5,50,100)`, s.OrganizationID, s.LocationID, id); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("PATCH", "/v1/admin/inventory/items/"+id+"/settings", strings.NewReader(`{"minimumStock":5,"optimalStock":10}`))
			req.SetPathValue("id", id)
			req = req.WithContext(context.WithValue(ctx, scopeKey{}, s))
			rec := httptest.NewRecorder()
			api.updateInventoryLocationSettings(rec, req)
			var reorder, optimal int
			if rec.Code != 204 {
				t.Fatalf("settings without reorder: %d %s", rec.Code, rec.Body.String())
			}
			if err := pool.QueryRow(ctx, `SELECT reorder_point::int,optimal_stock::int FROM inventory_location_settings WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, id).Scan(&reorder, &optimal); err != nil || reorder != 5 || optimal != 10 {
				t.Fatalf("legacy reorder still constrains optimal: reorder=%d optimal=%d err=%v", reorder, optimal, err)
			}
		}
		listReq := httptest.NewRequest("GET", "/v1/admin/inventory?q=Niveles", nil)
		listReq = listReq.WithContext(context.WithValue(ctx, scopeKey{}, s))
		listRec := httptest.NewRecorder()
		api.listInventory(listRec, listReq)
		var result struct {
			Items []struct {
				ID        string `json:"inventoryItemId"`
				Countable bool   `json:"unitCountable"`
			} `json:"items"`
		}
		if listRec.Code != 200 || json.Unmarshal(listRec.Body.Bytes(), &result) != nil {
			t.Fatalf("inventory: %d %s", listRec.Code, listRec.Body.String())
		}
		found := false
		for _, item := range result.Items {
			if item.ID == id {
				found = true
				if item.Countable != (unit == "botella") {
					t.Fatalf("unit metadata: %+v", item)
				}
			}
		}
		if !found {
			t.Fatal("created item not returned")
		}
	}
}
