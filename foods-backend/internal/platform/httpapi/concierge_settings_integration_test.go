package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConciergeEntitlementControlsPublicQRCode(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_WHATSAPP_PHONE", "+51987654321")

	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'whatsapp_bot',false)
		ON CONFLICT(organization_id,module_key)
		DO UPDATE SET active=false,updated_at=now()
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}

	var tableID, token string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tables(
		  organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled
		)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING id,qr_token
	`, s.OrganizationID, s.LocationID, fmt.Sprintf("Mesa entitlement %d", nonce)).Scan(&tableID, &token); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tables WHERE id=$1 AND organization_id=$2`, tableID, s.OrganizationID)
	})

	publicReq := httptest.NewRequest("GET", "/v1/public/tables/"+token, nil)
	publicReq.SetPathValue("token", token)
	publicRec := httptest.NewRecorder()
	api.getTableByQR(publicRec, publicReq)
	if publicRec.Code != 200 {
		t.Fatalf("public table before entitlement: %d %s", publicRec.Code, publicRec.Body.String())
	}
	var before struct {
		ConciergeEnabled bool   `json:"conciergeEnabled"`
		WhatsAppPhone    string `json:"whatsappPhone"`
	}
	if err := json.Unmarshal(publicRec.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	if before.ConciergeEnabled || before.WhatsAppPhone != "" {
		t.Fatalf("concierge must be unavailable before global activation: %#v", before)
	}

	statusReq := httptest.NewRequest("GET", "/v1/admin/concierge-settings", nil)
	statusReq = statusReq.WithContext(context.WithValue(statusReq.Context(), scopeKey{}, s))
	statusRec := httptest.NewRecorder()
	api.getConciergeSettings(statusRec, statusReq)
	if statusRec.Code != 200 {
		t.Fatalf("read concierge status: %d %s", statusRec.Code, statusRec.Body.String())
	}
	var disabled conciergeSettingsView
	if err := json.Unmarshal(statusRec.Body.Bytes(), &disabled); err != nil {
		t.Fatal(err)
	}
	if disabled.Active || disabled.Available || !disabled.ManagedByPlatform {
		t.Fatalf("unexpected disabled concierge status: %#v", disabled)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE organization_modules
		SET active=true,updated_at=now()
		WHERE organization_id=$1 AND module_key='whatsapp_bot'
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}

	publicAfterReq := httptest.NewRequest("GET", "/v1/public/tables/"+token, nil)
	publicAfterReq.SetPathValue("token", token)
	publicAfterRec := httptest.NewRecorder()
	api.getTableByQR(publicAfterRec, publicAfterReq)
	if publicAfterRec.Code != 200 {
		t.Fatalf("public table after entitlement: %d %s", publicAfterRec.Code, publicAfterRec.Body.String())
	}
	var after struct {
		ConciergeEnabled bool   `json:"conciergeEnabled"`
		WhatsAppPhone    string `json:"whatsappPhone"`
	}
	if err := json.Unmarshal(publicAfterRec.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if !after.ConciergeEnabled || after.WhatsAppPhone != "+51987654321" {
		t.Fatalf("public QR must expose the global Fudia WhatsApp after activation: %#v", after)
	}

	statusAfterReq := httptest.NewRequest("GET", "/v1/admin/concierge-settings", nil)
	statusAfterReq = statusAfterReq.WithContext(context.WithValue(statusAfterReq.Context(), scopeKey{}, s))
	statusAfterRec := httptest.NewRecorder()
	api.getConciergeSettings(statusAfterRec, statusAfterReq)
	if statusAfterRec.Code != 200 {
		t.Fatalf("read active concierge status: %d %s", statusAfterRec.Code, statusAfterRec.Body.String())
	}
	var enabled conciergeSettingsView
	if err := json.Unmarshal(statusAfterRec.Body.Bytes(), &enabled); err != nil {
		t.Fatal(err)
	}
	if !enabled.Active || !enabled.Available || enabled.WhatsAppPhone != "+51987654321" || !enabled.ManagedByPlatform {
		t.Fatalf("unexpected active concierge status: %#v", enabled)
	}
}
