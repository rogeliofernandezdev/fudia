package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestInventoryEditsArticleNameAndStockLevelsAtomically(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	call := func(actor scope, id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PATCH", "/v1/admin/inventory/items/"+id+"/settings", strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(ctx, scopeKey{}, actor))
		rec := httptest.NewRecorder()
		api.updateInventoryLocationSettings(rec, req)
		return rec
	}
	create := func(body string) inventoryProductOption {
		t.Helper()
		req := httptest.NewRequest("POST", "/v1/admin/purchase-inventory-items", strings.NewReader(body))
		req = req.WithContext(context.WithValue(ctx, scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.createPurchaseInventoryItem(rec, req)
		var item inventoryProductOption
		if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &item) != nil {
			t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
		}
		return item
	}
	ingredient := create(`{"newIngredient":{"name":"Papa blaca"},"unit":"kg"}`)
	category := seedRetailCategory(t, pool, s)
	product := create(fmt.Sprintf(`{"newProduct":{"name":"Agua minera","categoryId":%q,"price":"5"},"unit":"botella"}`, category))
	for _, item := range []inventoryProductOption{ingredient, product} {
		req := httptest.NewRequest("POST", "/v1/admin/inventory/entries", strings.NewReader(fmt.Sprintf(`{"inventoryItemId":%q,"unit":%q,"quantity":10,"unitCost":2}`, item.ID, item.Unit)))
		req = req.WithContext(context.WithValue(ctx, scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.createInventoryEntry(rec, req)
		if rec.Code != 201 {
			t.Fatalf("entry: %d %s", rec.Code, rec.Body.String())
		}
	}
	for _, row := range []struct {
		item inventoryProductOption
		name string
	}{{ingredient, "Papa blanca"}, {product, "Agua mineral"}} {
		rec := call(s, row.item.ID, fmt.Sprintf(`{"name":%q,"minimumStock":3,"optimalStock":8}`, row.name))
		if rec.Code != 204 {
			t.Fatalf("rename: %d %s", rec.Code, rec.Body.String())
		}
		var stored string
		var quantity, cost float64
		var movements int
		err := pool.QueryRow(ctx, `SELECT i.name,b.quantity::float8,b.average_unit_cost::float8,(SELECT count(*) FROM stock_movements m WHERE m.organization_id=i.organization_id AND m.inventory_item_id=i.id)
		  FROM inventory_items i JOIN stock_balances b ON b.organization_id=i.organization_id AND b.inventory_item_id=i.id AND b.location_id=$3 WHERE i.organization_id=$1 AND i.id=$2`, s.OrganizationID, row.item.ID, s.LocationID).Scan(&stored, &quantity, &cost, &movements)
		if err != nil || stored != row.name || quantity != 10 || cost != 2 || movements != 1 {
			t.Fatalf("identity/stock/history changed: name=%q qty=%v cost=%v movements=%d err=%v", stored, quantity, cost, movements, err)
		}
		if row.item.ProductID != nil {
			var commercial string
			if err = pool.QueryRow(ctx, `SELECT name FROM products WHERE organization_id=$1 AND id=$2`, s.OrganizationID, *row.item.ProductID).Scan(&commercial); err != nil || commercial != row.name {
				t.Fatalf("menu name not synchronized: %q %v", commercial, err)
			}
		}
		// Saving the current name must exclude this item and its linked product.
		if rec = call(s, row.item.ID, fmt.Sprintf(`{"name":%q,"minimumStock":4,"optimalStock":9}`, row.name)); rec.Code != 204 {
			t.Fatalf("self conflict: %d %s", rec.Code, rec.Body.String())
		}
	}
	for _, body := range []string{`{"name":"  PAPA   BLANCA  ","minimumStock":99,"optimalStock":100}`, `{"name":"","minimumStock":99}`, fmt.Sprintf(`{"name":%q}`, strings.Repeat("a", 161))} {
		rec := call(s, product.ID, body)
		if rec.Code != 409 && rec.Code != 400 {
			t.Fatalf("invalid rename: %d %s", rec.Code, rec.Body.String())
		}
	}
	var name string
	var minimum float64
	if err := pool.QueryRow(ctx, `SELECT p.name,l.minimum_stock::float8 FROM products p JOIN inventory_items i ON i.product_id=p.id AND i.organization_id=p.organization_id JOIN inventory_location_settings l ON l.inventory_item_id=i.id AND l.location_id=$3 WHERE i.organization_id=$1 AND i.id=$2`, s.OrganizationID, product.ID, s.LocationID).Scan(&name, &minimum); err != nil || name != "Agua mineral" || minimum != 4 {
		t.Fatalf("rejected rename changed data: %q %v %v", name, minimum, err)
	}
	if rec := call(other, product.ID, `{"name":"Ajeno","minimumStock":0}`); rec.Code != 404 {
		t.Fatalf("cross-company edit: %d %s", rec.Code, rec.Body.String())
	}
	invalidLocation := s
	invalidLocation.LocationID = other.LocationID
	if rec := call(invalidLocation, product.ID, `{"name":"Local ajeno","minimumStock":0}`); rec.Code != 404 {
		t.Fatalf("cross-location edit: %d %s", rec.Code, rec.Body.String())
	}
	// A failure after both names were updated must roll everything back.
	_, err := pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION test_reject_inventory_name_edit() RETURNS trigger LANGUAGE plpgsql AS $fn$ BEGIN IF NEW.inventory_item_id='%s'::uuid AND NEW.minimum_stock=99 THEN RAISE EXCEPTION 'fixture failure'; END IF; RETURN NEW; END $fn$;
	 CREATE TRIGGER test_reject_inventory_name_edit BEFORE INSERT OR UPDATE ON inventory_location_settings FOR EACH ROW EXECUTE FUNCTION test_reject_inventory_name_edit()`, product.ID))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS test_reject_inventory_name_edit ON inventory_location_settings; DROP FUNCTION IF EXISTS test_reject_inventory_name_edit()`)
	})
	if rec := call(s, product.ID, `{"name":"Cambio fallido","minimumStock":99,"optimalStock":100}`); rec.Code != 503 {
		t.Fatalf("forced rollback: %d %s", rec.Code, rec.Body.String())
	}
	var inventoryName, productName string
	if err = pool.QueryRow(ctx, `SELECT i.name,p.name FROM inventory_items i JOIN products p ON p.organization_id=i.organization_id AND p.id=i.product_id WHERE i.organization_id=$1 AND i.id=$2`, s.OrganizationID, product.ID).Scan(&inventoryName, &productName); err != nil || inventoryName != "Agua mineral" || productName != "Agua mineral" {
		t.Fatalf("rollback lost names: %q %q %v", inventoryName, productName, err)
	}
	// Different articles cannot acquire the same name concurrently.
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 2)
	var group sync.WaitGroup
	for _, id := range []string{ingredient.ID, product.ID} {
		group.Add(1)
		go func(itemID string) {
			defer group.Done()
			<-start
			responses <- call(s, itemID, `{"name":"Nombre concurrente","minimumStock":4,"optimalStock":9}`)
		}(id)
	}
	close(start)
	group.Wait()
	close(responses)
	saved := 0
	for rec := range responses {
		if rec.Code == 204 {
			saved++
		} else if rec.Code != 409 || !strings.Contains(rec.Body.String(), "inventory_item_name_taken") {
			t.Fatalf("concurrent edit: %d %s", rec.Code, rec.Body.String())
		}
	}
	if saved != 1 {
		t.Fatalf("concurrent renames succeeded %d times", saved)
	}
}
