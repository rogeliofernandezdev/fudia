package httpapi

import (
	"context"
	"testing"

	conciergeapp "github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	conciergedomain "github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	conciergepg "github.com/foods-platform/foods-backend/internal/modules/concierge/infrastructure/postgres"
	noticedomain "github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
	noticepg "github.com/foods-platform/foods-backend/internal/modules/notifications/infrastructure/postgres"
	platformapp "github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	platformpg "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

func TestWebChannelWithoutPhoneAndAnonymousNoticeOrigin(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	s := seedInventoryScope(t, pool)
	work := database.NewUnitOfWork(pool)
	platform := platformapp.New(platformpg.New(pool), work, nil, nil, nil, nil, nil).WithWhatsApp(&verifiedWhatsApp{state: "connected"})
	// Separate countries can have no number without violating the phone index.
	var channelIDs []string
	for _, country := range []string{"WX", "WY"} {
		if _, err := pool.Exec(ctx, `INSERT INTO platform_countries(code,name,default_currency) VALUES($1,$2,'PEN')`, country, "País de prueba web"); err != nil {
			t.Fatal(err)
		}
		channel, err := platform.CreateChannel(ctx, s, platformdomain.PlatformWhatsAppChannelInput{CountryCode: country, Provider: platformdomain.FudiaWeb, DisplayName: "Chat de prueba", Active: true})
		if err != nil {
			t.Fatal(err)
		}
		channelIDs = append(channelIDs, channel.ID)
		if channel.PhoneNumber != "" || channel.RoutingKey == "" {
			t.Fatal("web channel requires no fabricated phone")
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM customer_notices WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM concierge_conversation_origins WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM concierge_venue_visits WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(ctx, `DELETE FROM platform_whatsapp_channel_history WHERE channel_id=ANY($1::uuid[])`, channelIDs)
		_, _ = pool.Exec(ctx, `DELETE FROM platform_whatsapp_channels WHERE id=ANY($1::uuid[])`, channelIDs)
		_, _ = pool.Exec(ctx, `DELETE FROM platform_whatsapp_routes WHERE country_code IN ('WX','WY')`)
		_, _ = pool.Exec(ctx, `DELETE FROM platform_countries WHERE code IN ('WX','WY')`)
	})
	var routingKey string
	if err := pool.QueryRow(ctx, `SELECT routing_key FROM platform_whatsapp_channels WHERE id=$1`, channelIDs[0]).Scan(&routingKey); err != nil {
		t.Fatal(err)
	}
	identity := conciergedomain.CustomerIdentity{Provider: platformdomain.FudiaWeb, Subject: "browser-visitor-without-verified-phone"}
	if !identity.Valid() {
		t.Fatal("anonymous web identity rejected")
	}
	fakePhone := identity
	fakePhone.VerifiedPhone = "51912345678"
	if fakePhone.Valid() {
		t.Fatal("web browser can never assert a verified phone")
	}
	repo := conciergepg.NewUsage(pool)
	tx, err := work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	err = repo.RecordConversationOrigin(tx.Context(), conciergeapp.QRContext{Scope: s, CountryCode: "WX"}, conciergedomain.ConversationInput{ConversationID: "web-notice-conversation", ChannelKey: routingKey, Recipient: "browser-session-private-recipient", Identity: identity, ProviderKey: "web:WX"})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var orderID string
	if err = pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel) VALUES($1,$2,'mostrador') RETURNING id::text`, s.OrganizationID, s.LocationID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	notices := noticepg.New(pool)
	for _, conversation := range []string{"web-notice-conversation", "unregistered-origin"} {
		if err = notices.InsertCustomer(ctx, s, noticedomain.CustomerInput{Kind: noticedomain.CustomerOrderReady, OrderID: orderID, ConversationID: conversation, DedupeKey: conversation}); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	var recipient string
	if err = pool.QueryRow(ctx, `SELECT count(*),COALESCE(min(recipient),'') FROM customer_notices WHERE order_id=$1`, orderID).Scan(&count, &recipient); err != nil {
		t.Fatal(err)
	}
	if count != 1 || recipient != "browser-session-private-recipient" {
		t.Fatalf("anonymous notice must use registered origin: %d %q", count, recipient)
	}
}
