package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestEmprendeSubscriptionAndRoleCatalog(t *testing.T) {
	api, s, _ := seedIdentityAdministrator(t)
	ctx := context.Background()
	var planID string
	if err := api.db.QueryRow(ctx, `SELECT id FROM subscription_plans WHERE code='emprende'`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	tx, err := api.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = createOrganizationSubscription(ctx, tx, s.OrganizationID, s.UserID, planID, "monthly", true); err != nil {
		t.Fatal(err)
	}
	if _, err = seedOrganizationRoles(ctx, tx, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = api.db.Exec(ctx, `DELETE FROM organization_subscriptions WHERE organization_id=$1`, s.OrganizationID)
	})

	modules := api.activeModules(s.OrganizationID)
	if !modules["combos"] || !modules["inventario"] || !modules["compras"] || modules["recetas"] || modules["kardex"] {
		t.Fatalf("incorrect Emprende modules: %+v", modules)
	}
	catalog, err := api.loadRoleCatalog(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, menu := range []string{"combos", "inventario", "compras"} {
		if !slices.Contains(roleCatalogValues(catalog.Menus), menu) {
			t.Fatalf("Emprende cannot configure %s", menu)
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/admin/roles", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	api.listRoles(rec, req)
	if rec.Code != 200 {
		t.Fatalf("list roles %d %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []roleView `json:"items"`
	}
	if err = json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	foundAdmin := false
	for _, role := range response.Items {
		for _, menu := range role.MenuAccess {
			if !slices.Contains(roleCatalogValues(catalog.Menus), menu) {
				t.Errorf("role leaked menu %s", menu)
			}
		}
		for _, permission := range role.Permissions {
			if !slices.Contains(roleCatalogValues(catalog.Permissions), permission) {
				t.Errorf("role leaked permission %s", permission)
			}
		}
		if role.SystemKey != nil && *role.SystemKey == "administrator" {
			foundAdmin = true
			for _, menu := range []string{"combos", "inventario", "compras"} {
				if !slices.Contains(role.MenuAccess, menu) {
					t.Fatalf("company administrator missing %s", menu)
				}
			}
			for _, permission := range []string{"inventory.read", "inventory.manage", "purchases.read", "purchases.manage"} {
				if !slices.Contains(role.Permissions, permission) {
					t.Fatalf("company administrator missing %s", permission)
				}
			}
		}
	}
	if !foundAdmin {
		t.Fatal("administrator not seeded")
	}

	// Even an old/manual active flag cannot expose a module outside the plan.
	if _, err = api.db.Exec(ctx, `UPDATE organization_modules SET active=true WHERE organization_id=$1 AND module_key='recetas'`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if api.activeModules(s.OrganizationID)["recetas"] {
		t.Fatal("off-plan activation exposed recipes")
	}

	callSave := func(id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/v1/admin/roles", bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		api.saveRole(rec, req, id != "")
		return rec
	}
	denied := callSave("", `{"name":"Fuera del plan","description":"Prueba","menuAccess":["recetas"],"permissions":["recipes.manage"]}`)
	if denied.Code != 400 || !strings.Contains(denied.Body.String(), "role_outside_plan") {
		t.Fatalf("off-plan assignment allowed: %d %s", denied.Code, denied.Body.String())
	}
	created := callSave("", `{"name":"Gestor de menús","description":"Prueba","menuAccess":["combos"],"permissions":["menu.read","menu.manage"]}`)
	if created.Code != 201 {
		t.Fatalf("combos role rejected: %d %s", created.Code, created.Body.String())
	}

	var customID string
	if err = api.db.QueryRow(ctx, `INSERT INTO roles(organization_id,name,description,menu_access,permissions) VALUES($1,'Downgraded','Prueba',ARRAY['usuarios','recetas'],ARRAY['users.read','recipes.manage']) RETURNING id`, s.OrganizationID).Scan(&customID); err != nil {
		t.Fatal(err)
	}
	saved := callSave(customID, `{"name":"Downgraded","description":"Prueba","menuAccess":["usuarios"],"permissions":["users.read"]}`)
	if saved.Code != 200 {
		t.Fatalf("update %d %s", saved.Code, saved.Body.String())
	}
	var menus, permissions []string
	if err = api.db.QueryRow(ctx, `SELECT menu_access,permissions FROM roles WHERE id=$1`, customID).Scan(&menus, &permissions); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(menus, "recetas") || !slices.Contains(permissions, "recipes.manage") {
		t.Fatal("edit erased hidden previous-plan assignments")
	}
	if strings.Contains(saved.Body.String(), "recipes.manage") {
		t.Fatal("hidden assignment visible in save response")
	}

	req = httptest.NewRequest("GET", "/v1/admin/permission-catalog", nil).WithContext(context.WithValue(ctx, scopeKey{}, s))
	rec = httptest.NewRecorder()
	api.getPermissionCatalog(rec, req)
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "recipes.manage") || !strings.Contains(rec.Body.String(), "combos") || !strings.Contains(rec.Body.String(), "inventory.manage") || !strings.Contains(rec.Body.String(), "purchases.manage") {
		t.Fatalf("unscoped HTTP catalog: %d %s", rec.Code, rec.Body.String())
	}
	// The platform master retains the commercial catalog, not a tenant-restricted view.
	if _, err = api.db.Exec(ctx, `UPDATE users SET platform_admin=true WHERE id=$1`, s.UserID); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	api.getPermissionCatalog(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "purchases.manage") || !strings.Contains(rec.Body.String(), "kiosco") {
		t.Fatal("platform master catalog restricted")
	}
}

func TestEmprendeComboMigrationUpDown(t *testing.T) {
	pool := integrationPool(t)
	existing := seedInventoryScope(t, pool)
	missing := seedInventoryScope(t, pool)
	cancelled := seedInventoryScope(t, pool)
	enabled := seedInventoryScope(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // No fixture or catalog changes escape this test.
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DROP TABLE migration_074_module_backup`)
	exec(`UPDATE subscription_plans SET module_keys=array_remove(module_keys,'combos') WHERE code='emprende'`)
	var planID string
	if err = tx.QueryRow(ctx, `SELECT id FROM subscription_plans WHERE code='emprende'`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	for _, s := range []scope{existing, missing, cancelled, enabled} {
		if err = createOrganizationSubscription(ctx, tx, s.OrganizationID, s.UserID, planID, "monthly", true); err != nil {
			t.Fatal(err)
		}
	}
	exec(`DELETE FROM organization_modules WHERE organization_id=$1 AND module_key='combos'`, missing.OrganizationID)
	exec(`UPDATE organization_subscriptions SET status='cancelled' WHERE organization_id=$1`, cancelled.OrganizationID)
	exec(`UPDATE organization_modules SET active=true WHERE organization_id=$1 AND module_key='combos'`, enabled.OrganizationID)
	migration := func(direction string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000074_emprende_menus_combos."+direction+".sql"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	exec(migration("up"))
	var count int
	if err = tx.QueryRow(ctx, `SELECT cardinality(array_positions(module_keys,'combos')) FROM subscription_plans WHERE id=$1`, planID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("plan combo count %d err %v", count, err)
	}
	active := func(org string) bool {
		t.Helper()
		var v bool
		if err := tx.QueryRow(ctx, `SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='combos'`, org).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	if !active(existing.OrganizationID) || !active(missing.OrganizationID) || !active(enabled.OrganizationID) || active(cancelled.OrganizationID) {
		t.Fatal("migration changed wrong subscribers")
	}
	exec(migration("down"))
	if active(existing.OrganizationID) || !active(enabled.OrganizationID) {
		t.Fatal("rollback failed to restore original flags")
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM organization_modules WHERE organization_id=$1 AND module_key='combos'`, missing.OrganizationID).Scan(&count); err != nil || count != 0 {
		t.Fatal("rollback retained inserted module")
	}
	if err = tx.QueryRow(ctx, `SELECT cardinality(array_positions(module_keys,'combos')) FROM subscription_plans WHERE id=$1`, planID).Scan(&count); err != nil || count != 0 {
		t.Fatal("rollback retained plan addition")
	}
	// If combos already belonged to the plan, rollback must not remove it.
	exec(`UPDATE subscription_plans SET module_keys=array_append(module_keys,'combos') WHERE id=$1`, planID)
	exec(migration("up"))
	exec(migration("down"))
	if err = tx.QueryRow(ctx, `SELECT cardinality(array_positions(module_keys,'combos')) FROM subscription_plans WHERE id=$1`, planID).Scan(&count); err != nil || count != 1 {
		t.Fatal("rollback erased prior plan entitlement")
	}
}
