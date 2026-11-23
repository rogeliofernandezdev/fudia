package httpapi

import (
	"context"
	"strings"
	"testing"

	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
)

// Delivery lo administra Fudia (2026-11-23): el administrador de la empresa no
// abre la opción «Delivery» ni reintenta centrales, aunque tenga todos sus permisos.
func TestDeliveryAdministrationIsPlatformOnly(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	if _, err := api.db.Exec(context.Background(), `UPDATE roles SET system_key='administrator',name='Administrador de empresa',permissions=$2,menu_access=$3 WHERE id=$1`,
		role, identitydomain.CompanyAdministratorPermissions, identitydomain.CompanyAdministratorMenuAccess); err != nil {
		t.Fatal(err)
	}
	for _, key := range identitydomain.CompanyAdministratorMenuAccess {
		if key == "delivery" {
			t.Fatal("the company administrator menu must not include Delivery")
		}
	}
	call := companySession(t, api, s)
	order := "00000000-0000-0000-0000-000000000001"
	for _, route := range []struct{ method, path string }{
		{"GET", "/v1/admin/concierge-settings/delivery-fees"},
		{"PUT", "/v1/admin/concierge-settings/delivery-fees"},
		{"GET", "/v1/admin/delivery/orders"},
		{"GET", "/v1/admin/delivery/partners"},
		{"GET", "/v1/admin/delivery/failed-deliveries"},
		{"POST", "/v1/admin/orders/" + order + "/delivery-dispatch/retry"},
	} {
		rec := call(route.method, route.path, map[string]any{"tiers": []any{}})
		if rec.Code != 403 || !strings.Contains(rec.Body.String(), "platform_forbidden") {
			t.Fatalf("%s %s: %d %s", route.method, route.path, rec.Code, rec.Body.String())
		}
	}
}
