package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestProductListAvailableQuantityUsesLocalDayAndOrderConsumption(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='Pacific/Kiritimati' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	create := func(body string) string {
		t.Helper()
		rec := postProduct(api, s, body)
		if rec.Code != 201 {
			t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
		}
		var p product
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		return p.ID
	}
	portions := create(`{"name":"Ají","description":"Descripción conservada","price":"20","quantityControl":"portions","initialPortionQuantity":15}`)
	uncontrolled := create(`{"name":"Sin límite","price":"5"}`)
	inventory := create(`{"name":"Bebida","price":"5","quantityControl":"inventory","productType":"retail"}`)
	var inventoryID, otherLocation, orderID string
	if _, err := pool.Exec(ctx, `INSERT INTO inventory_units(organization_id,code,name) VALUES($1,'litros','Litros')`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO inventory_items(organization_id,sku,name,unit,product_id) VALUES($1,'LIST-STOCK','Bebida','litros',$2) RETURNING id`, s.OrganizationID, inventory).Scan(&inventoryID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,timezone) VALUES($1,'Otro local','OTHER','America/Lima') RETURNING id`, s.OrganizationID).Scan(&otherLocation); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity) VALUES($1,$2,$4,2.5),($1,$3,$4,99)`, s.OrganizationID, s.LocationID, otherLocation, inventoryID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity) VALUES($1,$2,$4,(now() AT TIME ZONE 'America/Lima')::date,90),($1,$3,$4,(now() AT TIME ZONE 'Pacific/Kiritimati')::date-1,80)`, s.OrganizationID, otherLocation, s.LocationID, portions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,created_by) VALUES($1,$2,'mostrador',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	consume := func(delta float64, kind string) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if err := api.applyOrderQuantityDelta(ctx, tx, s, orderID, map[string]float64{portions: delta}, kind); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	list := func(local string, query string) map[string]productListItem {
		t.Helper()
		current := s
		current.LocationID = local
		req := httptest.NewRequest("GET", "/?"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, current))
		rec := httptest.NewRecorder()
		api.listProducts(rec, req)
		if rec.Code != 200 {
			t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
		}
		var body struct {
			Items []productListItem `json:"items"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		result := map[string]productListItem{}
		for _, item := range body.Items {
			result[item.ID] = item
		}
		return result
	}
	assertQuantity := func(items map[string]productListItem, id string, expected float64) {
		t.Helper()
		p, ok := items[id]
		if !ok || p.AvailableQuantity == nil || *p.AvailableQuantity != expected {
			t.Fatalf("quantity for %s: %+v, expected %v", id, p, expected)
		}
	}
	items := list(s.LocationID, "")
	assertQuantity(items, portions, 15)
	assertQuantity(items, inventory, 2.5)
	if items[inventory].InventoryUnit == nil || *items[inventory].InventoryUnit != "litros" || items[uncontrolled].AvailableQuantity != nil || items[portions].Description != "Descripción conservada" {
		t.Fatalf("units, null control or preserved description: %+v", items)
	}
	consume(3, "sale")
	assertQuantity(list(s.LocationID, "q=Aj"), portions, 12)
	consume(-3, "reversal")
	assertQuantity(list(s.LocationID, ""), portions, 15)
	assertQuantity(list(otherLocation, ""), portions, 90)
	assertQuantity(list(otherLocation, ""), inventory, 99)
	if rec := updateAvailabilityRequest(t, api, s, portions, `{"status":"sold_out","reason":"Agotado en cocina"}`); rec.Code != 204 {
		t.Fatalf("mark exhausted: %d %s", rec.Code, rec.Body.String())
	}
	assertQuantity(list(s.LocationID, ""), portions, 0)
	if rec := updateAvailabilityRequest(t, api, s, portions, `{"status":"available","reason":"Producción repuesta"}`); rec.Code != 204 {
		t.Fatalf("reactivate: %d %s", rec.Code, rec.Body.String())
	}
	assertQuantity(list(s.LocationID, ""), portions, 15)
	for _, sql := range []string{
		`UPDATE products SET active=false WHERE id=$1`,
		`UPDATE products SET active=true,available_until=now()-interval '1 hour' WHERE id=$1`,
	} {
		if _, err := pool.Exec(ctx, sql, portions); err != nil {
			t.Fatal(err)
		}
		assertQuantity(list(s.LocationID, ""), portions, 0)
	}
	if _, err := pool.Exec(ctx, `UPDATE products SET available_until=NULL WHERE id=$1`, portions); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM product_availability WHERE product_id=$1 AND location_id=$2 AND business_date=(now() AT TIME ZONE 'Pacific/Kiritimati')::date`, portions, s.LocationID); err != nil {
		t.Fatal(err)
	}
	assertQuantity(list(s.LocationID, ""), portions, 0)
}
