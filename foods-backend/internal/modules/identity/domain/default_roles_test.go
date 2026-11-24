package domain

import (
	"slices"
	"testing"
)

func defaultRole(t *testing.T, key string) OrganizationRoleSeed {
	t.Helper()
	for _, role := range DefaultOrganizationRoles {
		if role.SystemKey == key {
			return role
		}
	}
	t.Fatalf("missing default role %s", key)
	return OrganizationRoleSeed{}
}

// Los roles predeterminados se nombran por la persona que los usa (2026-11-23).
func TestDefaultRolesAreNamedAfterThePerson(t *testing.T) {
	if got := defaultRole(t, "accounting").Name; got != "Contador" {
		t.Fatalf("accounting role name: %q", got)
	}
	if got := defaultRole(t, "buyer").Name; got != "Comprador" {
		t.Fatalf("buyer role name: %q", got)
	}
}

// El Contador consulta Ventas y Comprobantes en solo lectura: no toma pedidos,
// no cobra ni opera turnos.
func TestAccountantReadsSalesAndReceiptsOnly(t *testing.T) {
	role := defaultRole(t, "accounting")
	for _, menu := range []string{"dashboard", "ventas", "comprobantes", "caja", "fiscal", "medios_pago"} {
		if !slices.Contains(role.MenuAccess, menu) {
			t.Errorf("accountant must open %s: %v", menu, role.MenuAccess)
		}
	}
	for _, menu := range []string{"pos", "pedidos", "salon"} {
		if slices.Contains(role.MenuAccess, menu) {
			t.Errorf("accountant must not open %s", menu)
		}
	}
	if !slices.Contains(role.Permissions, "orders.read") {
		t.Fatalf("accountant needs orders.read for Ventas and Comprobantes: %v", role.Permissions)
	}
	for _, permission := range []string{"orders.manage", "cash.manage", "cash.registers.manage"} {
		if slices.Contains(role.Permissions, permission) {
			t.Errorf("accountant must not have %s", permission)
		}
	}
}
