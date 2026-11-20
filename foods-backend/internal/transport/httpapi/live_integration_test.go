package httpapi

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/platform/realtime"
)

// Un cambio confirmado en una tabla operativa llega al local por el canal en
// vivo, una sola vez por tema y transacción, y no llega a otro local.
func TestLiveChangesReachOnlyTheirLocationAfterCommit(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hub := realtime.NewHub()
	go realtime.Listen(ctx, pool, hub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	deadline := time.Now().Add(5 * time.Second)
	for !hub.Ready() {
		if time.Now().After(deadline) {
			t.Fatal("listener did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	topics, unsubscribe := hub.Subscribe(s.OrganizationID, s.LocationID)
	defer unsubscribe()
	foreign, unsubscribeForeign := hub.Subscribe(other.OrganizationID, other.LocationID)
	defer unsubscribeForeign()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err = tx.Exec(ctx, `INSERT INTO orders(organization_id,location_id,channel,status) VALUES($1,$2,'mostrador','nuevo')`, s.OrganizationID, s.LocationID); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case topic := <-topics:
		t.Fatalf("changes must not be announced before commit, got %q", topic)
	case <-time.After(150 * time.Millisecond):
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case topic := <-topics:
		if topic != "orders" {
			t.Fatalf("expected orders, got %q", topic)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("change did not arrive")
	}
	select {
	case topic := <-topics:
		t.Fatalf("one transaction must announce each topic once, got another %q", topic)
	case topic := <-foreign:
		t.Fatalf("another company must not receive %q", topic)
	case <-time.After(200 * time.Millisecond):
	}
}
