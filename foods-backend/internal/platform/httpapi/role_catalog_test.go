package httpapi

import (
	"slices"
	"strings"
	"testing"
)

func TestRoleCatalogFiltersContractedModulesAndDependencies(t *testing.T) {
	modules := map[string]bool{"reportes": true, "pos": true, "pedidos": true, "cocina": true, "mesas": true, "caja": true, "productos": true, "combos": true, "clientes": true, "locales": true, "fiscal": true, "usuarios": true}
	catalog := roleCatalogForModules(modules)
	menus, permissions := roleCatalogValues(catalog.Menus), roleCatalogValues(catalog.Permissions)
	for _, key := range []string{"dashboard", "disponibilidad", "combos", "usuarios"} {
		if !slices.Contains(menus, key) {
			t.Errorf("missing essential menu %s", key)
		}
	}
	for _, key := range []string{"recetas", "inventario", "compras", "reservas", "facturacion", "whatsapp_bot"} {
		if slices.Contains(menus, key) {
			t.Errorf("off-plan menu %s", key)
		}
	}
	for _, key := range []string{"recipes.manage", "inventory.read", "purchases.read", "reservations.read", "receipts.read", "expenses.read", "delivery.read"} {
		if slices.Contains(permissions, key) {
			t.Errorf("off-plan permission %s", key)
		}
	}
	if !slices.Contains(permissions, "menu.manage") || !slices.Contains(permissions, "subscription.read") {
		t.Fatal("essential capabilities missing")
	}
	modules["recetas"] = true
	if slices.Contains(roleCatalogValues(roleCatalogForModules(modules).Menus), "recetas") {
		t.Fatal("recipes requires inventory")
	}
	modules["inventario"] = true
	upgraded := roleCatalogForModules(modules)
	if !slices.Contains(roleCatalogValues(upgraded.Menus), "recetas") || !slices.Contains(roleCatalogValues(upgraded.Permissions), "recipes.manage") {
		t.Fatal("upgraded catalog missing recipes")
	}
}

func TestRoleCatalogProjectionMatchesVisibleCounts(t *testing.T) {
	catalog := roleCatalogForModules(map[string]bool{"usuarios": true, "combos": true})
	role := roleView{MenuAccess: []string{"usuarios", "combos", "compras"}, Permissions: []string{"users.read", "menu.manage", "purchases.read"}}
	projected := catalog.project(role)
	if len(projected.MenuAccess) != 2 || len(projected.Permissions) != 2 {
		t.Fatalf("misleading counts: %+v", projected)
	}
	if len(role.MenuAccess) != 3 || len(role.Permissions) != 3 {
		t.Fatal("projection modified stored assignment")
	}
	wildcard := catalog.project(roleView{MenuAccess: []string{"*"}, Permissions: []string{"*"}})
	if !slices.Equal(wildcard.MenuAccess, roleCatalogValues(catalog.Menus)) || !slices.Equal(wildcard.Permissions, roleCatalogValues(catalog.Permissions)) {
		t.Fatal("wildcard leaked outside effective catalog")
	}
	master := effectiveRoleCatalog{Menus: menuAccessCatalog, Permissions: permissionCatalog, Unrestricted: true}
	if len(master.project(role).MenuAccess) != 3 {
		t.Fatal("platform catalog unexpectedly filtered")
	}
	// Filtering must not mutate the shared source catalogs.
	if !slices.Contains(roleCatalogValues(menuAccessCatalog), "compras") {
		t.Fatal("shared catalog mutated")
	}
}

func TestEveryCatalogPermissionHasModuleOwnership(t *testing.T) {
	for _, key := range roleCatalogValues(permissionCatalog) {
		prefix, _, _ := strings.Cut(key, ".")
		if key != "subscription.read" && len(rolePermissionModules[prefix]) == 0 {
			t.Errorf("unmapped permission %s", key)
		}
	}
	if !permissionAvailableForModules("inventory.read", map[string]bool{"kardex": true}) {
		t.Fatal("kardex read missing")
	}
	if permissionAvailableForModules("inventory.manage", map[string]bool{"kardex": true}) {
		t.Fatal("kardex granted inventory mutation")
	}
}
