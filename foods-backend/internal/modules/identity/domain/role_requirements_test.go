package domain

import (
	"slices"
	"testing"
)

func TestCompleteRolePermissionsAddsWhatScreensAndActionsNeed(t *testing.T) {
	all := func(string) bool { return true }
	got := CompleteRolePermissions([]string{"recetas"}, []string{"recipes.manage"}, all)
	for _, want := range []string{"recipes.manage", "menu.read", "inventory.read"} {
		if !slices.Contains(got, want) {
			t.Errorf("recipes must include %s: %v", want, got)
		}
	}
	// Transitivo: administrar usuarios -> ver usuarios -> ver locales.
	got = CompleteRolePermissions(nil, []string{"users.manage"}, all)
	if !slices.Contains(got, "users.read") || !slices.Contains(got, "organizations.read") {
		t.Fatalf("users.manage chain: %v", got)
	}
	// Sin duplicados y respetando el orden de lo marcado.
	got = CompleteRolePermissions([]string{"pedidos"}, []string{"orders.read", "orders.manage"}, all)
	if got[0] != "orders.read" || got[1] != "orders.manage" || len(got) != 3 || got[2] != "menu.read" {
		t.Fatalf("orders: %v", got)
	}
	// Fuera del plan no se agrega.
	got = CompleteRolePermissions([]string{"recetas"}, []string{"recipes.manage"}, func(p string) bool { return p != "inventory.read" })
	if slices.Contains(got, "inventory.read") || !slices.Contains(got, "menu.read") {
		t.Fatalf("off-plan requirement must not be added: %v", got)
	}
}

func TestEveryRequirementIsACatalogValue(t *testing.T) {
	menus, permissions := RoleCatalogValues(MenuAccessCatalog), RoleCatalogValues(PermissionCatalog)
	for menu, required := range MenuRequirements {
		if !slices.Contains(menus, menu) {
			t.Errorf("unknown menu %s", menu)
		}
		for _, permission := range required {
			if !slices.Contains(permissions, permission) {
				t.Errorf("menu %s requires unknown %s", menu, permission)
			}
		}
	}
	for permission, required := range PermissionRequirements {
		if !slices.Contains(permissions, permission) {
			t.Errorf("unknown permission %s", permission)
		}
		for _, next := range required {
			if !slices.Contains(permissions, next) {
				t.Errorf("%s requires unknown %s", permission, next)
			}
		}
	}
}

// Los roles predefinidos ya cumplen sus propios requisitos.
func TestDefaultRolesSatisfyRequirements(t *testing.T) {
	for _, role := range DefaultOrganizationRoles {
		completed := CompleteRolePermissions(role.MenuAccess, role.Permissions, func(string) bool { return true })
		if len(completed) != len(role.Permissions) {
			t.Errorf("%s is missing %v", role.Name, completed[len(role.Permissions):])
		}
	}
}
