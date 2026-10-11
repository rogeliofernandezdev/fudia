package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryUnitCatalogAndAuthorization(t *testing.T) {
	pool := integrationPool(t)
	s, other := seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	ctx := context.Background()
	api := New(pool)
	call := func(s scope, method, body string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/v1/admin/purchase-units", strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	list := call(s, "GET", "", api.listInventoryUnits)
	var catalog struct {
		Items []inventoryUnitView `json:"items"`
	}
	if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &catalog) != nil {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	for _, code := range []string{"bolsa", "paquete"} {
		found := false
		for _, unit := range catalog.Items {
			if unit.Code == code {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing seeded %s", code)
		}
	}
	custom := `{"code":" SACO-25 ","name":"Saco de 25 kg"}`
	created := call(s, "POST", custom, api.createInventoryUnit)
	var unit inventoryUnitView
	if created.Code != 201 || json.Unmarshal(created.Body.Bytes(), &unit) != nil || unit.Code != "saco-25" || unit.ID == "" {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	if rec := call(s, "POST", custom, api.createInventoryUnit); rec.Code != 409 {
		t.Fatalf("duplicate: %d", rec.Code)
	}
	if rec := call(s, "POST", `{"code":"unit space","name":"Invalid"}`, api.createInventoryUnit); rec.Code != 400 {
		t.Fatalf("invalid: %d", rec.Code)
	}
	if rec := call(other, "GET", "", api.listInventoryUnits); strings.Contains(rec.Body.String(), "saco-25") {
		t.Fatal("catalog crossed organization")
	}
	rec := call(other, "POST", `{"newIngredient":{"name":"Foreign"},"unit":"saco-25"}`, api.createPurchaseInventoryItem)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "invalid_inventory_unit") {
		t.Fatalf("foreign unit accepted: %d %s", rec.Code, rec.Body.String())
	}
	rec = call(s, "POST", `{"newIngredient":{"name":"Arroz"},"unit":"saco-25"}`, api.createPurchaseInventoryItem)
	if rec.Code != 201 {
		t.Fatalf("custom article: %d %s", rec.Code, rec.Body.String())
	}
	var savedUnit string
	if err := pool.QueryRow(ctx, `SELECT unit FROM inventory_items WHERE organization_id=$1 AND name='Arroz'`, s.OrganizationID).Scan(&savedUnit); err != nil || savedUnit != unit.Code {
		t.Fatalf("persisted unit: %s %v", savedUnit, err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND action='inventory_unit.created' AND entity_id=$2`, s.OrganizationID, unit.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit: %d %v", count, err)
	}
	if rec := call(s, "POST", `{"newIngredient":{"name":"Invalid stock"},"quantity":2,"unit":"missing"}`, api.createInventoryEntry); rec.Code != 400 {
		t.Fatalf("entry unit: %d %s", rec.Code, rec.Body.String())
	}

	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Units reader',ARRAY['purchases.read']) RETURNING id`, s.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "inventory-unit-test-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := api.Routes()
	route := func(method, path string, auth bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(`{"code":"fardo","name":"Fardo"}`))
		if auth {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
	if rec := route("GET", "/v1/admin/purchase-units", false); rec.Code != 401 {
		t.Fatalf("anonymous: %d", rec.Code)
	}
	if rec := route("GET", "/v1/admin/purchase-units", true); rec.Code != 200 {
		t.Fatalf("reader: %d %s", rec.Code, rec.Body.String())
	}
	if rec := route("POST", "/v1/admin/purchase-units", true); rec.Code != 403 {
		t.Fatalf("reader write: %d", rec.Code)
	}
	if rec := route("GET", "/v1/admin/purchase-combinations", true); rec.Code != 200 {
		t.Fatalf("combination reader %d %s", rec.Code, rec.Body.String())
	}
	if rec := route("POST", "/v1/admin/purchase-combinations", true); rec.Code != 403 {
		t.Fatalf("combination reader write %d", rec.Code)
	}
	if rec := route("POST", "/v1/admin/purchase-inventory-items/00000000-0000-0000-0000-000000000000/presentations", true); rec.Code != 403 {
		t.Fatalf("presentation reader write %d", rec.Code)
	}
	if rec := route("GET", "/v1/admin/inventory/units", true); rec.Code != 403 {
		t.Fatalf("inventory access: %d", rec.Code)
	}
	if _, err := pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['purchases.read','purchases.manage','inventory.read'] WHERE id=$1`, roleID); err != nil {
		t.Fatal(err)
	}
	if rec := route("POST", "/v1/admin/purchase-units", true); rec.Code != 201 {
		t.Fatalf("manager: %d %s", rec.Code, rec.Body.String())
	}
	if rec := route("GET", "/v1/admin/inventory/units", true); rec.Code != 200 || !strings.Contains(rec.Body.String(), "fardo") {
		t.Fatalf("shared catalog: %d %s", rec.Code, rec.Body.String())
	}
	if rec := route("POST", "/v1/admin/inventory/units", true); rec.Code != 403 {
		t.Fatalf("inventory manager: %d", rec.Code)
	}
}

func TestInventoryUnitsMigrationPreservesLegacyStock(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	latestMigration, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000077_inventory_combinations.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, string(latestMigration)); err != nil {
		t.Fatal(err)
	}
	migration := func(direction string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000076_inventory_units."+direction+".sql"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, string(data)); err != nil {
			t.Fatal(err)
		}
	}
	migration("down")
	var itemID string
	if err = tx.QueryRow(ctx, `INSERT INTO inventory_items(organization_id,sku,name,unit) VALUES($1,'LEGACY-UNIT','Legacy','sacos antiguos') RETURNING id`, s.OrganizationID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity) VALUES($1,$2,$3,4.5)`, s.OrganizationID, s.LocationID, itemID); err != nil {
		t.Fatal(err)
	}
	migration("up")
	var unit string
	var quantity float64
	if err = tx.QueryRow(ctx, `SELECT ii.unit,sb.quantity::float8 FROM inventory_items ii JOIN stock_balances sb ON sb.inventory_item_id=ii.id WHERE ii.id=$1`, itemID).Scan(&unit, &quantity); err != nil || unit != "sacos antiguos" || quantity != 4.5 {
		t.Fatalf("altered legacy stock: %s %v %v", unit, quantity, err)
	}
	var catalogName string
	if err = tx.QueryRow(ctx, `SELECT name FROM inventory_units WHERE organization_id=$1 AND code=$2`, s.OrganizationID, unit).Scan(&catalogName); err != nil || catalogName != unit {
		t.Fatalf("legacy unit missing: %s %v", catalogName, err)
	}
	migration("down")
	if err = tx.QueryRow(ctx, `SELECT unit FROM inventory_items WHERE id=$1`, itemID).Scan(&unit); err != nil || unit != "sacos antiguos" {
		t.Fatalf("rollback lost unit: %s %v", unit, err)
	}
}

func TestInventoryUnitAuditFailureRollsBack(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `CREATE FUNCTION fail_unit_audit_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='inventory_unit.created' THEN RAISE EXCEPTION 'audit unavailable in test'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_unit_audit_test BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION fail_unit_audit_test()`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DROP TRIGGER fail_unit_audit_test ON audit_log; DROP FUNCTION fail_unit_audit_test()`)
	})
	req := httptest.NewRequest("POST", "/v1/admin/purchase-units", strings.NewReader(`{"code":"rollback-unit","name":"Rollback unit"}`))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	New(pool).createInventoryUnit(rec, req)
	if rec.Code != 503 {
		t.Fatalf("audit failure: %d %s", rec.Code, rec.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_units WHERE organization_id=$1 AND code='rollback-unit'`, s.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial unit persisted: %d %v", count, err)
	}
}
