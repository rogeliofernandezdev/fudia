package httpapi

import (
	"context"
	"strings"
)

// Technical ownership of permissions, not a second subscription catalog.
var rolePermissionModules = map[string][]string{
	"dashboard": {"reportes"}, "users": {"usuarios"},
	"organizations": {"locales", "fiscal", "whatsapp_bot"}, "fiscal": {"fiscal"},
	"menu": {"productos", "combos", "recetas"}, "recipes": {"recetas"}, "customers": {"clientes"},
	"orders": {"pedidos", "pos", "cocina", "reportes"}, "kitchen": {"cocina"}, "bar": {"cocina"},
	"tables": {"mesas"}, "reservations": {"reservas"}, "cash": {"caja", "pos"},
	"receipts": {"facturacion"}, "inventory": {"inventario"}, "purchases": {"compras"},
	"expenses": {"costos"}, "reports": {"reportes"}, "audit": {"reportes"},
	"delivery": {"delivery", "repartidores"},
}

type effectiveRoleCatalog struct {
	Menus, Permissions []map[string]any
	Unrestricted       bool
}

func permissionAvailableForModules(permission string, modules map[string]bool) bool {
	// Subscription is a company-level account capability, not an optional module.
	if permission == "subscription.read" {
		return true
	}
	if permission == "inventory.read" && modules["kardex"] {
		return true
	}
	prefix, _, _ := strings.Cut(permission, ".")
	for _, module := range rolePermissionModules[prefix] {
		if modules[module] && (module != "recetas" || modules["inventario"]) {
			return true
		}
	}
	return false
}

func filterRoleCatalog(groups []map[string]any, allowed func(string) bool) []map[string]any {
	result := []map[string]any{}
	for _, group := range groups {
		items := []map[string]string{}
		for _, item := range group["items"].([]map[string]string) {
			if allowed(item["value"]) {
				items = append(items, item)
			}
		}
		if len(items) > 0 {
			result = append(result, map[string]any{"group": group["group"], "items": items})
		}
	}
	return result
}

func roleCatalogForModules(modules map[string]bool) effectiveRoleCatalog {
	menus := filterRoleCatalog(menuAccessCatalog, func(key string) bool {
		switch key {
		case "dashboard":
			key = "reportes"
		case "disponibilidad":
			key = "productos"
		}
		return modules[key] && (key != "recetas" || modules["inventario"])
	})
	permissions := filterRoleCatalog(permissionCatalog, func(key string) bool {
		return permissionAvailableForModules(key, modules)
	})
	return effectiveRoleCatalog{Menus: menus, Permissions: permissions}
}

func (a *API) loadRoleCatalog(ctx context.Context, s scope) (effectiveRoleCatalog, error) {
	var platform bool
	err := a.db.QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active AND (organization_id=$2 OR platform_admin)`, s.UserID, s.OrganizationID).Scan(&platform)
	if err != nil {
		return effectiveRoleCatalog{}, err
	}
	if platform {
		return effectiveRoleCatalog{Menus: menuAccessCatalog, Permissions: permissionCatalog, Unrestricted: true}, nil
	}
	modules, err := a.readOrganizationModules(ctx, s.OrganizationID)
	if err != nil {
		return effectiveRoleCatalog{}, err
	}
	return roleCatalogForModules(modules), nil
}

func roleCatalogValues(groups []map[string]any) []string {
	values := []string{}
	for _, group := range groups {
		for _, item := range group["items"].([]map[string]string) {
			values = append(values, item["value"])
		}
	}
	return values
}

func effectiveRoleValues(values []string, groups []map[string]any) []string {
	allowed := roleCatalogValues(groups)
	selected := map[string]bool{}
	for _, value := range values {
		selected[value] = true
	}
	result := []string{}
	for _, value := range allowed {
		if selected["*"] || selected[value] {
			result = append(result, value)
		}
	}
	return result
}

func (catalog effectiveRoleCatalog) project(role roleView) roleView {
	if !catalog.Unrestricted {
		role.MenuAccess = effectiveRoleValues(role.MenuAccess, catalog.Menus)
		role.Permissions = effectiveRoleValues(role.Permissions, catalog.Permissions)
	}
	return role
}
