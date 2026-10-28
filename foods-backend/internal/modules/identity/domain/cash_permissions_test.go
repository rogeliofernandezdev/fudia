package domain

import "testing"

func TestDefaultCashAdministrationBelongsOnlyToCompanyAdministrator(t *testing.T) {
	for _, role := range DefaultOrganizationRoles {
		administrative := false
		operational := false
		for _, permission := range role.Permissions {
			administrative = administrative || permission == "cash.registers.manage"
			operational = operational || permission == "cash.manage"
		}
		if administrative != (role.SystemKey == "administrator") {
			t.Fatalf("unexpected register administration for %s", role.SystemKey)
		}
		if role.SystemKey == "cashier" && !operational {
			t.Fatal("cashier lost shift operation")
		}
	}
	if !PermissionAvailableForModules("cash.registers.manage", map[string]bool{"caja": true}) {
		t.Fatal("permission missing from Caja catalog")
	}
}
