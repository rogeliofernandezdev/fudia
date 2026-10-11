package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPurchaseCombinationsCatalogConversionsAndDefault(t *testing.T) {
	pool := integrationPool(t)
	s, other := seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	call := func(who scope, method, path, body, id string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.SetPathValue("id", id)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, who))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	list := call(s, "GET", "/v1/admin/purchase-combinations?unit=botella", "", "", api.listInventoryCombinations)
	if list.Code != 200 {
		t.Fatal(list.Body.String())
	}
	var catalog struct{ Items []inventoryCombinationView }
	if json.Unmarshal(list.Body.Bytes(), &catalog) != nil {
		t.Fatal("invalid catalog")
	}
	defaultCount := 0
	for _, row := range catalog.Items {
		if row.Unit != "botella" {
			t.Fatal("filter crossed unit")
		}
		if row.IsDefault {
			defaultCount++
			if row.Code != "package" {
				t.Fatal("bottle default not package")
			}
		}
	}
	if defaultCount != 1 {
		t.Fatalf("defaults %d", defaultCount)
	}
	for _, rec := range []*httptest.ResponseRecorder{
		call(s, "POST", "/v1/admin/purchase-combinations", `{"unit":"botella","code":"crate","name":"Jaba","isDefault":true}`, "", api.createInventoryCombination),
		call(s, "POST", "/v1/admin/purchase-combinations", `{"unit":"lata","code":"crate","name":"Jaba"}`, "", api.createInventoryCombination),
	} {
		if rec.Code != 201 {
			t.Fatalf("combination: %d %s", rec.Code, rec.Body.String())
		}
	}
	foreign := call(other, "GET", "/v1/admin/purchase-combinations", "", "", api.listInventoryCombinations)
	if strings.Contains(foreign.Body.String(), "Jaba") {
		t.Fatal("tenant catalog leaked")
	}
	if rec := call(s, "POST", "/v1/admin/purchase-combinations", `{"unit":"missing","code":"new","name":"Missing"}`, "", api.createInventoryCombination); rec.Code != 400 {
		t.Fatalf("invalid unit %d", rec.Code)
	}
	if rec := call(s, "POST", "/v1/admin/purchase-combinations", `{"unit":"botella","code":"crate","name":"Another"}`, "", api.createInventoryCombination); rec.Code != 409 {
		t.Fatalf("rename reused code %d", rec.Code)
	}
	category := seedRetailCategory(t, pool, s)
	body := fmt.Sprintf(`{"unit":"botella","newProduct":{"name":"Agua combinaciones","price":"2.50","categoryId":%q},"presentations":[{"presentationType":"package","unitsPerPresentation":10,"isDefault":true},{"presentationType":"box","unitsPerPresentation":24}]}`, category)
	created := call(s, "POST", "/v1/admin/purchase-inventory-items", body, "", api.createPurchaseInventoryItem)
	var item inventoryProductOption
	if created.Code != 201 || json.Unmarshal(created.Body.Bytes(), &item) != nil {
		t.Fatalf("article: %d %s", created.Code, created.Body.String())
	}
	if item.Quantity != "0" || len(item.Presentations) != 3 {
		t.Fatalf("initial stock/presentations: %#v", item)
	}
	var packageID string
	for _, p := range item.Presentations {
		if p.IsDefault {
			if p.PresentationType != "package" || p.Name != "Paquete" {
				t.Fatalf("default %#v", p)
			}
			packageID = p.ID
		}
	}
	if packageID == "" {
		t.Fatal("default missing")
	}
	// An existing article can gain a new factor/default without moving stock or changing prior factors.
	saved := call(s, "POST", "/v1/admin/purchase-inventory-items/"+item.ID+"/presentations", `{"presentationType":"crate","unitsPerPresentation":12,"isDefault":true}`, item.ID, api.savePurchasePresentation)
	if saved.Code != 201 {
		t.Fatalf("save presentation %d %s", saved.Code, saved.Body.String())
	}
	if rec := call(other, "POST", "/v1/admin/purchase-inventory-items/"+item.ID+"/presentations", `{"presentationType":"unit","unitsPerPresentation":1}`, item.ID, api.savePurchasePresentation); rec.Code != 404 {
		t.Fatalf("foreign item %d", rec.Code)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_presentations WHERE organization_id=$1 AND inventory_item_id=$2 AND is_default`, s.OrganizationID, item.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("defaults %d %v", count, err)
	}
	// A forbidden unit/type pairing must roll back the article, not just its presentation.
	rejected := call(other, "POST", "/v1/admin/purchase-inventory-items", `{"unit":"botella","newIngredient":{"name":"Forbidden combination"},"presentations":[{"presentationType":"crate","unitsPerPresentation":12}]}`, "", api.createPurchaseInventoryItem)
	if rejected.Code != 400 {
		t.Fatalf("invalid pairing %d %s", rejected.Code, rejected.Body.String())
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_items WHERE organization_id=$1 AND name='Forbidden combination'`, other.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial article %d %v", count, err)
	}
	// Purchase and receive five packages => fifty bottles, never five inventory units.
	var supplierID string
	if err := pool.QueryRow(ctx, `INSERT INTO suppliers(organization_id,name) VALUES($1,'Combinations supplier') RETURNING id`, s.OrganizationID).Scan(&supplierID); err != nil {
		t.Fatal(err)
	}
	ordered := call(s, "POST", "/v1/admin/purchase-orders", fmt.Sprintf(`{"supplierId":%q,"items":[{"inventoryItemId":%q,"presentationId":%q,"quantity":5,"unitCost":"10.00"}]}`, supplierID, item.ID, packageID), "", api.createPurchaseOrder)
	var order struct{ ID string }
	if ordered.Code != 201 || json.Unmarshal(ordered.Body.Bytes(), &order) != nil {
		t.Fatalf("order %d %s", ordered.Code, ordered.Body.String())
	}
	var lineID string
	if _, err := pool.Exec(ctx, `UPDATE purchase_orders SET status='approved',approved_at=now() WHERE id=$1`, order.ID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM purchase_order_items WHERE purchase_order_id=$1`, order.ID).Scan(&lineID); err != nil {
		t.Fatal(err)
	}
	received := call(s, "POST", "/v1/admin/purchase-orders/"+order.ID+"/receive", fmt.Sprintf(`{"idempotencyKey":"combination-receipt","items":[{"purchaseOrderItemId":%q,"quantity":5}]}`, lineID), order.ID, api.receivePurchaseOrder)
	if received.Code != 201 {
		t.Fatalf("receive %d %s", received.Code, received.Body.String())
	}
	var balance float64
	if err := pool.QueryRow(ctx, `SELECT quantity::float8 FROM stock_balances WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, item.ID).Scan(&balance); err != nil || balance != 50 {
		t.Fatalf("conversion balance %v %v", balance, err)
	}
	detail := call(s, "GET", "/v1/admin/purchase-orders/"+order.ID, "", order.ID, api.getPurchaseOrder)
	if detail.Code != 200 || !strings.Contains(detail.Body.String(), `"presentationName":"Paquete"`) {
		t.Fatalf("catalog label %d %s", detail.Code, detail.Body.String())
	}
	// Historical factor stays 10, even after a new preferred crate of 12.
	var factor float64
	if err := pool.QueryRow(ctx, `SELECT units_per_presentation::float8 FROM purchase_order_items WHERE id=$1`, lineID).Scan(&factor); err != nil || factor != 10 {
		t.Fatalf("history changed %v %v", factor, err)
	}
	// Selling one bottle uses the inventory unit, never the preferred crate factor.
	var saleID string
	if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,created_by) VALUES($1,$2,'mostrador',$3) RETURNING id`, s.OrganizationID, s.LocationID, s.UserID).Scan(&saleID); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if quantityErr := api.applyOrderQuantityDelta(ctx, tx, s, saleID, map[string]float64{*item.ProductID: 1}, "sale"); quantityErr != nil {
		t.Fatal(quantityErr.Message)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT quantity::float8 FROM stock_balances WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, item.ID).Scan(&balance); err != nil || balance != 49 {
		t.Fatalf("sale deducted packaging instead of unit: %v %v", balance, err)
	}
	// Five bags of half a kilogram add 2.5 kg.
	entry := call(s, "POST", "/v1/admin/inventory/entries", `{"newIngredient":{"name":"Arroz medio kilo"},"unit":"kg","quantity":5,"presentationType":"bag","unitsPerPresentation":0.5}`, "", api.createInventoryEntry)
	if entry.Code != 201 {
		t.Fatalf("fractional contents %d %s", entry.Code, entry.Body.String())
	}
	var converted struct{ Balance float64 }
	if json.Unmarshal(entry.Body.Bytes(), &converted) != nil || converted.Balance != 2.5 {
		t.Fatalf("fractional stock %s", entry.Body.String())
	}
	// Restore base unit as the catalog default; it must be exposed exactly once.
	rec := call(s, "POST", "/v1/admin/purchase-combinations", `{"unit":"botella","code":"unit","name":"Unidad base","isDefault":true}`, "", api.createInventoryCombination)
	if rec.Code != 201 {
		t.Fatal(rec.Body.String())
	}
	rec = call(s, "GET", "/v1/admin/purchase-combinations?unit=botella", "", "", api.listInventoryCombinations)
	catalog.Items = nil
	_ = json.Unmarshal(rec.Body.Bytes(), &catalog)
	defaultCount = 0
	for _, row := range catalog.Items {
		if row.IsDefault {
			defaultCount++
			if row.Code != "unit" {
				t.Fatal("default base unit not selected")
			}
		}
	}
	if defaultCount != 1 {
		t.Fatalf("base defaults %d", defaultCount)
	}
}

func TestInventoryCombinationAuditFailureRollsBack(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `CREATE FUNCTION fail_combination_audit_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='inventory_combination.saved' THEN RAISE EXCEPTION 'audit unavailable in test'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_combination_audit_test BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_combination_audit_test()`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DROP TRIGGER fail_combination_audit_test ON audit_log; DROP FUNCTION fail_combination_audit_test()`)
	})
	req := httptest.NewRequest("POST", "/v1/admin/purchase-combinations", strings.NewReader(`{"unit":"botella","code":"rollback","name":"Rollback","isDefault":true}`))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	New(pool).createInventoryCombination(rec, req)
	if rec.Code != 503 {
		t.Fatalf("audit %d", rec.Code)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_presentation_types WHERE organization_id=$1 AND code='rollback'`, s.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial catalog %d %v", count, err)
	}
}

func TestInventoryCombinationsMigrationPreservesConversions(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	migration := func(direction string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000077_inventory_combinations."+direction+".sql"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, string(data)); err != nil {
			t.Fatal(err)
		}
	}
	migration("down")
	var item, id string
	if err = tx.QueryRow(ctx, `INSERT INTO inventory_items(organization_id,name,sku,unit) VALUES($1,'Legacy package','LEGACY-COMB','botella') RETURNING id`, s.OrganizationID).Scan(&item); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO inventory_presentations(organization_id,inventory_item_id,presentation_type,units_per_presentation) VALUES($1,$2,'package',10) RETURNING id`, s.OrganizationID, item).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity) VALUES($1,$2,$3,50)`, s.OrganizationID, s.LocationID, item); err != nil {
		t.Fatal(err)
	}
	migration("up")
	var factor, balance float64
	var preferred bool
	if err = tx.QueryRow(ctx, `SELECT p.units_per_presentation::float8,p.is_default,sb.quantity::float8 FROM inventory_presentations p JOIN stock_balances sb ON sb.inventory_item_id=p.inventory_item_id WHERE p.id=$1`, id).Scan(&factor, &preferred, &balance); err != nil || factor != 10 || balance != 50 || !preferred {
		t.Fatalf("changed migration %v %v %v %v", factor, balance, preferred, err)
	}
	migration("down")
	migration("up")
}
