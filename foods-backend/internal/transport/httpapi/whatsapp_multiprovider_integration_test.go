package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	conciergeapp "github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	conciergedomain "github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	conciergepg "github.com/foods-platform/foods-backend/internal/modules/concierge/infrastructure/postgres"
	platformapp "github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	platformpg "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type verifiedWhatsApp struct{ state string }

func (v *verifiedWhatsApp) Verify(_ context.Context, in platformdomain.PlatformWhatsAppChannelInput) (platformdomain.WhatsAppConnection, error) {
	key := "authorized-meta-asset"
	if in.Provider == platformdomain.FudiaWeb {
		key = "web:" + in.CountryCode
	}
	if in.Provider == platformdomain.Whatsmeow {
		key = "authorized-linked-device"
	}
	return platformdomain.WhatsAppConnection{Provider: in.Provider, CountryCode: in.CountryCode, PhoneNumber: in.PhoneNumber, ProviderKey: key, State: v.state}, nil
}
func (v *verifiedWhatsApp) Link(context.Context, platformdomain.WhatsAppLinkInput) (platformdomain.WhatsAppConnection, error) {
	return platformdomain.WhatsAppConnection{Provider: platformdomain.Whatsmeow, State: v.state}, nil
}

func TestWhatsAppCountryConfigurationAndOriginalConversationRoutes(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	second := seedInventoryScope(t, pool)
	ctx := context.Background()
	country := "QW"
	if _, err := pool.Exec(ctx, `INSERT INTO platform_countries(code,name,default_currency) VALUES($1,'País de prueba WhatsApp','PEN') ON CONFLICT(code) DO UPDATE SET active=true`, country); err != nil {
		t.Fatal(err)
	}
	provider := &verifiedWhatsApp{state: "connected"}
	work := database.NewUnitOfWork(pool)
	platform := platformapp.New(platformpg.New(pool), work, nil, nil, nil, nil, nil).WithWhatsApp(provider)
	in := platformdomain.PlatformWhatsAppChannelInput{CountryCode: country, PhoneNumber: "+51976543210", DisplayName: "Fudia prueba", Active: true}
	channel, err := platform.CreateChannel(ctx, actor, in)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM concierge_conversation_origins WHERE routing_key IN(SELECT routing_key FROM platform_whatsapp_routes WHERE country_code=$1)`, country)
		_, _ = pool.Exec(ctx, `DELETE FROM concierge_venue_visits WHERE organization_id=ANY($1::uuid[])`, []string{actor.OrganizationID, second.OrganizationID})
		_, _ = pool.Exec(ctx, `DELETE FROM platform_whatsapp_channel_history WHERE channel_id=$1`, channel.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM platform_whatsapp_channels WHERE id=$1`, channel.ID)
		_, _ = pool.Exec(ctx, `DELETE FROM platform_whatsapp_routes WHERE country_code=$1`, country)
	})
	if channel.Provider != platformdomain.MetaCloudAPI || channel.Revision != 1 {
		t.Fatalf("Meta default: %+v", channel)
	}
	if _, err = platform.CreateChannel(ctx, actor, in); err == nil {
		t.Fatal("duplicate country accepted")
	}
	legacy := platformdomain.WhatsAppRoute{CountryCode: country, Provider: platformdomain.MetaCloudAPI, ProviderKey: "123456789", PhoneNumber: channel.PhoneNumber}
	if err = platform.RegisterLegacyRoute(ctx, legacy); err != nil {
		t.Fatalf("legacy import: %v", err)
	}
	legacy.PhoneNumber = "+51933333333"
	if err = platform.RegisterLegacyRoute(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	var legacyPhone string
	if err = pool.QueryRow(ctx, `SELECT phone_number FROM platform_whatsapp_routes WHERE routing_key=$1`, legacy.ProviderKey).Scan(&legacyPhone); err != nil || legacyPhone != channel.PhoneNumber {
		t.Fatal("legacy route was rewritten")
	}
	originalKey := channel.RoutingKey
	provider.state = "disconnected"
	in.Provider = platformdomain.Whatsmeow
	in.PhoneNumber = "+51987654321"
	in.Revision = &channel.Revision
	if _, err = platform.UpdateChannel(ctx, actor, channel.ID, in); err == nil {
		t.Fatal("disconnected provider activated")
	}
	old, err := platformpg.New(pool).Channel(ctx, channel.ID)
	if err != nil || old.Provider != platformdomain.MetaCloudAPI || old.RoutingKey != originalKey {
		t.Fatal("failed verification changed current config")
	}
	// Visit two real tenants and pin the original routing key before the change.
	identity := conciergedomain.CustomerIdentity{Provider: platformdomain.MetaCloudAPI, Subject: "51912345678", VerifiedPhone: "51912345678"}
	repo := conciergepg.New(pool)
	origins := conciergepg.NewUsage(pool)
	recordVisit := func(s scope, conversation string, id conciergedomain.CustomerIdentity, namespace, key string) {
		t.Helper()
		tx, err := work.Begin(ctx, unitofwork.Options{})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		err = origins.RecordConversationOrigin(tx.Context(), conciergeapp.QRContext{Scope: s, CountryCode: country}, conciergedomain.ConversationInput{ConversationID: conversation, ChannelKey: key, Recipient: id.Subject, Identity: id, ProviderKey: namespace})
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	suffix := time.Now().Format("150405.000000000")
	conversation := "visit-a-" + suffix
	for _, s := range []scope{actor, second} {
		if _, err = pool.Exec(ctx, `UPDATE organization_fiscal_profiles SET country_code=$2 WHERE organization_id=$1`, s.OrganizationID, country); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'whatsapp_bot',true) ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true`, s.OrganizationID); err != nil {
			t.Fatal(err)
		}
	}
	recordVisit(actor, conversation, identity, "authorized-meta-asset", originalKey)
	recordVisit(actor, conversation, identity, "authorized-meta-asset", originalKey) // Retry does not count another visit.
	recordVisit(second, "visit-b-"+suffix, identity, "authorized-meta-asset", originalKey)
	list := func(id conciergedomain.CustomerIdentity, namespace string) conciergedomain.RestaurantPage {
		t.Helper()
		tx, err := work.Begin(ctx, unitofwork.Options{})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		page, err := repo.RestaurantDirectory(tx.Context(), country, namespace, id, true, 0)
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		return page
	}
	page := list(identity, "authorized-meta-asset")
	if len(page.Items) != 2 || page.Items[0].VisitCount != 1 || page.Items[0].OrganizationID == page.Items[1].OrganizationID {
		t.Fatalf("visits mixed or retried: %+v", page)
	}
	// The customer notice captures its origin on INSERT, before number changes.
	var order, notice string
	if err = pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,customer_phone) VALUES($1,$2,'whatsapp',$3) RETURNING id::text`, actor.OrganizationID, actor.LocationID, identity.VerifiedPhone).Scan(&order); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO customer_notices(organization_id,location_id,order_id,kind,customer_phone,conversation_id) VALUES($1,$2,$3,'order_ready',$4,$5) RETURNING id::text`, actor.OrganizationID, actor.LocationID, order, identity.VerifiedPhone, conversation).Scan(&notice); err != nil {
		t.Fatal(err)
	}
	provider.state = "connected"
	updated, err := platform.UpdateChannel(ctx, actor, channel.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != channel.ID || updated.Provider != platformdomain.Whatsmeow || updated.Revision != 2 || updated.RoutingKey == originalKey {
		t.Fatalf("not an in-place provider change: %+v", updated)
	}
	if _, err = platform.UpdateChannel(ctx, actor, channel.ID, in); err == nil {
		t.Fatal("stale configuration overwritten")
	}
	routes, err := platform.WhatsAppRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	foundOld, foundNew := false, false
	for _, route := range routes {
		if route.RoutingKey == originalKey {
			foundOld = !route.Active && route.PhoneNumber == channel.PhoneNumber
		}
		if route.RoutingKey == updated.RoutingKey {
			foundNew = route.Active
		}
	}
	if !foundOld || !foundNew {
		t.Fatal("historical route lost or current route missing")
	}
	var pinned, recipient string
	if err = pool.QueryRow(ctx, `SELECT routing_key,recipient FROM customer_notices WHERE id=$1`, notice).Scan(&pinned, &recipient); err != nil || pinned != originalKey || recipient != identity.Subject {
		t.Fatal("pending notice moved to new number")
	}
	history, err := platform.ChannelHistory(ctx, channel.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("audit history: %d %v", len(history), err)
	}
	public, _ := json.Marshal(updated)
	if strings.Contains(string(public), "authorized-linked-device") || strings.Contains(string(public), "routingKey") {
		t.Fatal("private provider reference exposed")
	}
	contextView, err := repo.RestaurantContext(ctx, country, actor.LocationID)
	if err != nil || len(contextView.QRToken) != 32 {
		t.Fatalf("selected venue: %+v %v", contextView, err)
	}
	again, err := repo.RestaurantContext(ctx, country, actor.LocationID)
	if err != nil || again.QRToken != contextView.QRToken {
		t.Fatal("selection replaced printed QR")
	}
	// A LID without provider proof never inherits somebody else's PN history.
	anonymous := conciergedomain.CustomerIdentity{Provider: platformdomain.Whatsmeow, Subject: fmt.Sprintf("%d@lid", time.Now().UnixNano())}
	if len(list(anonymous, "authorized-linked-device").Items) != 0 {
		t.Fatal("LID inferred as telephone")
	}
	recordVisit(actor, "lid-visit-"+suffix, anonymous, "authorized-linked-device", updated.RoutingKey)
	anonymous.VerifiedPhone = identity.VerifiedPhone
	if len(list(anonymous, "authorized-linked-device").Items) != 2 {
		t.Fatal("verified alias did not recover persisted visits")
	}
	anonymous.VerifiedPhone = "51999999999"
	tx, _ := work.Begin(ctx, unitofwork.Options{})
	defer tx.Rollback(ctx)
	if _, err = repo.RestaurantDirectory(tx.Context(), country, "authorized-linked-device", anonymous, true, 0); err == nil {
		t.Fatal("verified alias rebound to a different person")
	}
}

func TestWhatsAppManagementRequiresGlobalAuthorization(t *testing.T) {
	pool := integrationPool(t)
	api := New(pool)
	handler := api.Routes()
	for _, path := range []string{"/v1/platform/whatsapp-channels", "/v1/platform/whatsapp-channels/link", "/v1/integrations/concierge/whatsapp/routes"} {
		method := "GET"
		if strings.HasSuffix(path, "/link") {
			method = "POST"
		}
		request := httptest.NewRequest(method, path, strings.NewReader(`{"countryCode":"PE"}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 401 && response.Code != 503 {
			t.Fatalf("unauthorized %s: %d", path, response.Code)
		}
	}
	// An authenticated restaurant user also cannot read, link or mutate the
	// platform configuration, even when the payload itself is valid.
	actor := seedInventoryScope(t, pool)
	call := companySession(t, api, actor)
	id := "00000000-0000-4000-8000-000000000001"
	for _, target := range []struct{ method, path string }{
		{"GET", "/v1/platform/whatsapp-channels"},
		{"POST", "/v1/platform/whatsapp-channels"},
		{"POST", "/v1/platform/whatsapp-channels/link"},
		{"PATCH", "/v1/platform/whatsapp-channels/" + id},
		{"GET", "/v1/platform/whatsapp-channels/" + id + "/history"},
	} {
		response := call(target.method, target.path, map[string]any{"countryCode": "PE", "provider": "WHATSMEOW", "phoneNumber": "+51912345678", "displayName": "Canal", "active": true})
		if response.Code != 403 {
			t.Fatalf("restaurant user %s: %d %s", target.path, response.Code, response.Body.String())
		}
	}
}
