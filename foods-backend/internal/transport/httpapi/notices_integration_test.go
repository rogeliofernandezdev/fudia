package httpapi

import (
	"context"
	"testing"

	noticespg "github.com/foods-platform/foods-backend/internal/modules/notifications/infrastructure/postgres"
)

// La campana solo lista avisos pendientes: comanda lista hasta entregarse, pedido nuevo
// hasta que la cocina lo marque listo, cuenta pedida hasta completarse; un pedido
// cancelado sale siempre. Los llamados siguen visibles.
func TestNoticeFeedHidesAttendedOrders(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	repo := noticespg.New(pool)

	order := func(status string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,total,created_by) VALUES($1,$2,'recojo',$3,10,$4) RETURNING id`,
			s.OrganizationID, s.LocationID, status, s.UserID).Scan(&id); err != nil {
			t.Fatalf("seed order: %v", err)
		}
		return id
	}
	notice := func(kind, orderID, title string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO staff_notices(organization_id,location_id,kind,audience,title,order_id) VALUES($1,$2,$3,'all',$4,NULLIF($5,'')::uuid)`,
			s.OrganizationID, s.LocationID, kind, title, orderID); err != nil {
			t.Fatalf("seed notice: %v", err)
		}
	}
	ready, delivered, fresh, cooked, cancelled, billOpen, billPaid := order("listo"), order("entregado"), order("confirmado"), order("listo"), order("cancelado"), order("entregado"), order("entregado")
	if _, err := pool.Exec(ctx, `UPDATE orders SET completed_at=now() WHERE id=$1`, billPaid); err != nil {
		t.Fatalf("complete order: %v", err)
	}
	notice("order_ready", ready, "listo pendiente")
	notice("order_ready", delivered, "listo entregado")
	notice("pickup_order", delivered, "nuevo entregado")
	notice("pickup_order", fresh, "nuevo pendiente")
	notice("pickup_order", cooked, "nuevo ya listo")
	notice("order_ready", cooked, "listo del nuevo")
	notice("order_ready", cancelled, "listo cancelado")
	notice("bill_requested", billOpen, "cuenta pendiente")
	notice("bill_requested", billPaid, "cuenta pagada")
	notice("assistance_requested", "", "llamado")

	feed, err := repo.Feed(ctx, s, []string{"all"}, 30)
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	got := map[string]bool{}
	for _, item := range feed.Items {
		got[item.Title] = true
	}
	for _, title := range []string{"listo pendiente", "nuevo pendiente", "listo del nuevo", "cuenta pendiente", "llamado"} {
		if !got[title] {
			t.Errorf("%q debe seguir en la campana", title)
		}
	}
	for _, title := range []string{"listo entregado", "nuevo entregado", "nuevo ya listo", "listo cancelado", "cuenta pagada"} {
		if got[title] {
			t.Errorf("%q ya fue atendido y no debe listarse", title)
		}
	}
	if feed.Unread != 5 || len(feed.Items) != 5 {
		t.Fatalf("pendientes = %d items / %d sin leer, want 5/5", len(feed.Items), feed.Unread)
	}
}
