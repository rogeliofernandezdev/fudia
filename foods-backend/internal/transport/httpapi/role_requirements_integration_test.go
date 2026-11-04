package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
)

// Un rol personalizado guarda también las acciones que exigen sus pantallas y
// acciones marcadas (Recetas necesita ver carta e inventario; administrar
// usuarios necesita ver usuarios y locales). El catálogo publica esos requisitos.
func TestRoleSaveCompletesRequiredPermissions(t *testing.T) {
	api, s, _ := seedIdentityAdministrator(t)
	ctx := context.Background()
	for _, module := range []string{"usuarios", "locales", "pedidos", "productos"} {
		if _, err := api.db.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,$2,true)
			ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()`, s.OrganizationID, module); err != nil {
			t.Fatal(err)
		}
	}
	save := func(body string) map[string]any {
		t.Helper()
		req := httptest.NewRequest("POST", "/v1/admin/roles", bytes.NewReader([]byte(body)))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		api.createRole(rec, req)
		var out map[string]any
		if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &out) != nil {
			t.Fatalf("create role: %d %s", rec.Code, rec.Body.String())
		}
		return out
	}
	stored := func(id string) []string {
		t.Helper()
		var permissions []string
		if err := api.db.QueryRow(ctx, `SELECT permissions FROM roles WHERE id=$1`, id).Scan(&permissions); err != nil {
			t.Fatal(err)
		}
		return permissions
	}

	// Usuarios y roles: ver usuarios exige ver locales para asignarlos.
	users := save(`{"name":"Encargado de personal","description":"Prueba","menuAccess":["usuarios"],"permissions":["users.manage"]}`)
	for _, want := range []string{"users.manage", "users.read", "organizations.read"} {
		if !slices.Contains(stored(users["id"].(string)), want) {
			t.Errorf("users role must include %s: %v", want, stored(users["id"].(string)))
		}
	}
	// Comanda: gestionar pedidos exige ver pedidos y la carta.
	orders := save(`{"name":"Mozo de apoyo","description":"Prueba","menuAccess":["pedidos"],"permissions":["orders.manage"]}`)
	for _, want := range []string{"orders.manage", "orders.read", "menu.read"} {
		if !slices.Contains(stored(orders["id"].(string)), want) {
			t.Errorf("orders role must include %s: %v", want, stored(orders["id"].(string)))
		}
	}

	// El catálogo publica los requisitos para que el editor los muestre.
	req := httptest.NewRequest("GET", "/v1/admin/permission-catalog", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.getPermissionCatalog(rec, req)
	var catalog struct {
		Requirements struct {
			Menus       map[string][]string `json:"menus"`
			Permissions map[string][]string `json:"permissions"`
		} `json:"requirements"`
	}
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &catalog) != nil {
		t.Fatalf("catalog: %d %s", rec.Code, rec.Body.String())
	}
	// Solo lo del plan: esta empresa no tiene Recetas ni Inventario.
	if !slices.Contains(catalog.Requirements.Menus["usuarios"], "organizations.read") || !slices.Contains(catalog.Requirements.Permissions["users.read"], "organizations.read") ||
		catalog.Requirements.Menus["recetas"] != nil || catalog.Requirements.Permissions["recipes.manage"] != nil {
		t.Fatalf("catalog requirements: %+v", catalog.Requirements)
	}
}

// La migración 000108 completa los roles personalizados que ya existían.
func TestRoleRequirementsBackfillCompletesExistingRoles(t *testing.T) {
	api, s, _ := seedIdentityAdministrator(t)
	ctx := context.Background()
	var id string
	if err := api.db.QueryRow(ctx, `
		INSERT INTO roles(organization_id,name,description,menu_access,permissions,active)
		VALUES($1,'Cocina antigua','Prueba',ARRAY['recetas'],ARRAY['recipes.manage'],true) RETURNING id`, s.OrganizationID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../../migrations/000108_role_permission_requirements.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	var permissions []string
	if err := api.db.QueryRow(ctx, `SELECT permissions FROM roles WHERE id=$1`, id).Scan(&permissions); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"recipes.manage", "menu.read", "inventory.read"} {
		if !slices.Contains(permissions, want) {
			t.Errorf("backfilled role must include %s: %v", want, permissions)
		}
	}
	// Idempotente: correrla otra vez no duplica acciones.
	if _, err := api.db.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("backfill again: %v", err)
	}
	var again []string
	if err := api.db.QueryRow(ctx, `SELECT permissions FROM roles WHERE id=$1`, id).Scan(&again); err != nil {
		t.Fatal(err)
	}
	if len(again) != len(permissions) {
		t.Fatalf("backfill must be idempotent: %v -> %v", permissions, again)
	}
}

// Moneda, decimales e impuesto los necesita cualquier usuario del panel: un mesero
// sin «Ver empresa y locales» los lee en la ruta real (antes recibía 403).
func TestOrgSettingsReadableByAnyAuthenticatedUser(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	var roleID string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Mesero ajustes',ARRAY['orders.read','tables.read']) RETURNING id`, s.OrganizationID).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, roleID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "settings-test-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := New(pool).Routes()
	call := func(method string, authenticated bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/v1/admin/settings", strings.NewReader(`{}`))
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
	if rec := call("GET", false); rec.Code != 401 {
		t.Fatalf("anonymous settings: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("GET", true); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"currencySymbol"`) {
		t.Fatalf("waiter must read settings: %d %s", rec.Code, rec.Body.String())
	}
	// Cambiarlos sigue exigiendo administrar la empresa.
	if rec := call("PATCH", true); rec.Code != 403 {
		t.Fatalf("waiter must not change settings: %d %s", rec.Code, rec.Body.String())
	}
}
