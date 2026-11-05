package domain

import (
	"slices"
	"testing"
)

// Cada acción por pantalla existe en el catálogo y cada acción del catálogo se
// presenta en alguna pantalla: no quedan casillas que no habiliten nada.
func TestScreenActionsCoverTheCatalog(t *testing.T) {
	menus, permissions := RoleCatalogValues(MenuAccessCatalog), RoleCatalogValues(PermissionCatalog)
	shown := map[string]bool{}
	for menu, actions := range ScreenActions {
		if !slices.Contains(menus, menu) {
			t.Errorf("unknown menu %s", menu)
		}
		seen := map[string]bool{}
		for _, action := range actions {
			if !slices.Contains(permissions, action.Value) {
				t.Errorf("%s shows unknown permission %s", menu, action.Value)
			}
			if seen[action.Value] {
				t.Errorf("%s repeats %s", menu, action.Value)
			}
			if action.Label == "" {
				t.Errorf("%s/%s has no label", menu, action.Value)
			}
			seen[action.Value], shown[action.Value] = true, true
		}
	}
	for _, permission := range permissions {
		if !shown[permission] {
			t.Errorf("permission %s is not shown in any screen", permission)
		}
	}
}

// Lo que una pantalla exige para abrirse aparece dentro de esa pantalla.
func TestScreenActionsIncludeMenuRequirements(t *testing.T) {
	for menu, required := range MenuRequirements {
		values := []string{}
		for _, action := range ScreenActions[menu] {
			values = append(values, action.Value)
		}
		for _, permission := range required {
			if !slices.Contains(values, permission) {
				t.Errorf("%s requires %s but does not show it", menu, permission)
			}
		}
	}
}

func TestScreenActionsForFollowsThePlan(t *testing.T) {
	catalog := RoleCatalogForModules(map[string]bool{"facturacion": true, "pedidos": true})
	got := catalog.ScreenActionsFor()
	if _, ok := got["caja"]; ok {
		t.Fatal("off-plan screen must be omitted")
	}
	if len(got["comprobantes"]) != 2 || got["comprobantes"][1].Label != "Reenviar comprobantes a SUNAT" {
		t.Fatalf("unexpected comprobantes actions: %#v", got["comprobantes"])
	}
}
