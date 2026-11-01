package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// The location form saves the location and its own tip settings in one
// transaction: a rejected tip configuration also discards the location change.
func TestLocationFormSavesTipSettingsAtomically(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',permissions=ARRAY['organizations.read','organizations.manage'],menu_access=ARRAY['locales'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	var planID string
	if err := api.db.QueryRow(ctx, `
		INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,module_keys,terms_version,active,max_locations,max_users)
		VALUES($1,'Plan locales','Plan de prueba','PEN',169,1690,0,ARRAY['locales'],'test-v1',true,5,10)
		RETURNING id`, fmt.Sprintf("location-tips-%d", time.Now().UnixNano())).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(ctx, `
		INSERT INTO organization_subscriptions(organization_id,plan_id,billing_cycle,price_amount,currency,status,current_period_starts_at,current_period_ends_at,renews_at)
		VALUES($1,$2,'monthly',169,'PEN','active',now(),now()+interval '30 days',now()+interval '30 days')`, s.OrganizationID, planID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = api.db.Exec(context.Background(), `DELETE FROM organization_subscriptions WHERE organization_id=$1`, s.OrganizationID)
		_, _ = api.db.Exec(context.Background(), `DELETE FROM subscription_plans WHERE id=$1`, planID)
	})
	call := companySession(t, api, s)
	var profileID string
	if err := api.db.QueryRow(ctx, `SELECT fiscal_profile_id FROM locations WHERE id=$1`, s.LocationID).Scan(&profileID); err != nil {
		t.Fatal(err)
	}
	editable := func() bool {
		t.Helper()
		w := call("GET", "/v1/admin/tip-settings", nil)
		var body struct {
			Editable bool `json:"editable"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
			t.Fatalf("tip settings: %d %s", w.Code, w.Body.String())
		}
		return body.Editable
	}
	tipMode := func(locationID string) string {
		t.Helper()
		var mode string
		err := api.db.QueryRow(ctx, `SELECT COALESCE((SELECT mode FROM tip_settings WHERE organization_id=$1 AND location_id=$2),'none')`, s.OrganizationID, locationID).Scan(&mode)
		if err != nil {
			t.Fatal(err)
		}
		return mode
	}
	locationName := func(id string) string {
		t.Helper()
		var name string
		if err := api.db.QueryRow(ctx, `SELECT name FROM locations WHERE id=$1`, id).Scan(&name); err != nil {
			t.Fatal(err)
		}
		return name
	}
	collect := map[string]any{"mode": "collect", "percentage": 10, "showSuggestion": true, "allowFixed": true, "channels": []string{"salon"}}
	location := func(name string, tips any) map[string]any {
		body := map[string]any{"name": name, "code": "TIPSFORM", "address": "Av. Larco 123", "phone": "", "openingHours": "", "timezone": "America/Lima", "fiscalProfileId": profileID}
		if tips != nil {
			body["tipSettings"] = tips
		}
		return body
	}

	if !editable() {
		t.Fatal("the reserved administrator must be able to edit tips")
	}

	// A new location is created with its tips in the same request.
	created := call("POST", "/v1/admin/locations", location("Sede propinas", collect))
	var out struct {
		ID string `json:"id"`
	}
	if created.Code != 201 || json.Unmarshal(created.Body.Bytes(), &out) != nil || out.ID == "" {
		t.Fatalf("create location with tips: %d %s", created.Code, created.Body.String())
	}
	if mode := tipMode(out.ID); mode != "collect" {
		t.Fatalf("tips not saved with the new location: %s", mode)
	}

	// Invalid tips reject the whole save.
	invalid := map[string]any{"mode": "collect", "percentage": 150, "channels": []string{"salon"}}
	if w := call("PATCH", "/v1/admin/locations/"+out.ID, location("Nombre que no debe quedar", invalid)); w.Code != 400 {
		t.Fatalf("invalid tips: %d %s", w.Code, w.Body.String())
	}
	if name := locationName(out.ID); name != "Sede propinas" {
		t.Fatalf("location changed despite rejected tips: %s", name)
	}

	// Without tipSettings the location saves and keeps its tips.
	if w := call("PATCH", "/v1/admin/locations/"+out.ID, location("Sede renombrada", nil)); w.Code != 204 {
		t.Fatalf("update without tips: %d %s", w.Code, w.Body.String())
	}
	if name, mode := locationName(out.ID), tipMode(out.ID); name != "Sede renombrada" || mode != "collect" {
		t.Fatalf("update without tips: %s %s", name, mode)
	}

	// Managing locations is not enough to change tips.
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='cashier' WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	if editable() {
		t.Fatal("only the reserved administrator may edit tips")
	}
	disabled := map[string]any{"mode": "disabled", "percentage": 10, "channels": []string{"salon"}}
	if w := call("PATCH", "/v1/admin/locations/"+out.ID, location("Cambio sin permiso", disabled)); w.Code != 403 {
		t.Fatalf("tips permission bypass: %d %s", w.Code, w.Body.String())
	}
	if name, mode := locationName(out.ID), tipMode(out.ID); name != "Sede renombrada" || mode != "collect" {
		t.Fatalf("forbidden tips must roll back the location: %s %s", name, mode)
	}
}
