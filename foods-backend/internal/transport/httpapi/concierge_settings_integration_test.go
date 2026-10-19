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
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")

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
		ConciergeEnabled bool `json:"conciergeEnabled"`
	}
	if err := json.Unmarshal(publicRec.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	if before.ConciergeEnabled {
		t.Fatalf("concierge must be unavailable before global activation: %#v", before)
	}
	var publicBody map[string]any
	if err := json.Unmarshal(publicRec.Body.Bytes(), &publicBody); err != nil {
		t.Fatal(err)
	}
	if _, exists := publicBody["whatsappPhone"]; exists {
		t.Fatalf("public table must not expose an app-stored WhatsApp number: %#v", publicBody)
	}

	blockedReq := httptest.NewRequest("GET", "/v1/integrations/concierge/"+token+"/menu", nil)
	blockedReq.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
	blockedRec := httptest.NewRecorder()
	api.Routes().ServeHTTP(blockedRec, blockedReq)
	if blockedRec.Code != 404 {
		t.Fatalf("concierge must reject interaction before module activation: %d %s", blockedRec.Code, blockedRec.Body.String())
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
	if disabled.Active || !disabled.ManagedByPlatform {
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
		ConciergeEnabled bool `json:"conciergeEnabled"`
	}
	if err := json.Unmarshal(publicAfterRec.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if !after.ConciergeEnabled {
		t.Fatalf("public QR must expose Concierge availability after activation: %#v", after)
	}

	allowedReq := httptest.NewRequest("GET", "/v1/integrations/concierge/"+token+"/menu", nil)
	allowedReq.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
	allowedRec := httptest.NewRecorder()
	api.Routes().ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != 200 {
		t.Fatalf("concierge must allow interaction after module activation: %d %s", allowedRec.Code, allowedRec.Body.String())
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
	if !enabled.Active || !enabled.ManagedByPlatform {
		t.Fatalf("unexpected active concierge status: %#v", enabled)
	}
	var statusBody map[string]any
	if err := json.Unmarshal(statusAfterRec.Body.Bytes(), &statusBody); err != nil {
		t.Fatal(err)
	}
	if _, exists := statusBody["whatsappPhone"]; exists {
		t.Fatalf("admin status must not expose a duplicated WhatsApp number: %#v", statusBody)
	}
}

// La carta digital del QR (Concierge) marca los productos recomendados y los presenta primero.
func TestConciergeMenuListsRecommendedFirst(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_modules(organization_id,module_key,active)
		VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var token string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING qr_token
	`, s.OrganizationID, s.LocationID, fmt.Sprintf("Mesa recomendados %d", nonce)).Scan(&token); err != nil {
		t.Fatal(err)
	}
	for _, product := range []struct {
		name     string
		featured bool
	}{{"Alfa normal", false}, {"Zeta recomendado", true}} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO products(organization_id,sku,name,price,active,product_type,quantity_control,featured,service_destination)
			VALUES($1,$2,$3,10,true,'prepared','none',$4,'kitchen')
		`, s.OrganizationID, fmt.Sprintf("REC-%d-%t", nonce, product.featured), product.name, product.featured); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest("GET", "/v1/integrations/concierge/"+token+"/menu", nil)
	req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
	rec := httptest.NewRecorder()
	api.Routes().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("menu: %d %s", rec.Code, rec.Body.String())
	}
	var menu struct {
		Items []struct {
			Name        string `json:"name"`
			Recommended bool   `json:"recommended"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &menu); err != nil {
		t.Fatal(err)
	}
	if len(menu.Items) < 2 || menu.Items[0].Name != "Zeta recomendado" || !menu.Items[0].Recommended || menu.Items[1].Recommended {
		t.Fatalf("recommended products must come first and be marked: %+v", menu.Items)
	}
}
