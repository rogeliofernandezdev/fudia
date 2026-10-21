package httpapi

import (
	"context"
	"os"
	"testing"
)

func TestDeliveryCompletionMigrationPreservesClosedHistoryAndOpenTables(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	// A temporary shadow table lets us exercise both scripts without changing
	// the application's migrated orders or other integration fixtures.
	if _, err = conn.Exec(ctx, `
		CREATE TEMP TABLE orders (
		  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
		  organization_id uuid NOT NULL,
		  location_id uuid NOT NULL,
		  table_id uuid,
		  channel text NOT NULL,
		  status text NOT NULL,
		  created_at timestamptz NOT NULL DEFAULT now(),
		  updated_at timestamptz NOT NULL DEFAULT now()
		);
		CREATE UNIQUE INDEX orders_one_open_table_uq ON orders(organization_id,location_id,table_id)
		  WHERE table_id IS NOT NULL AND status NOT IN ('entregado','cancelado');
		CREATE INDEX orders_open_location_idx ON orders(organization_id,location_id,created_at DESC)
		  WHERE status NOT IN ('entregado','cancelado');
		CREATE INDEX orders_open_table_lookup_idx ON orders(organization_id,location_id,table_id,created_at DESC)
		  WHERE table_id IS NOT NULL AND status NOT IN ('entregado','cancelado');
		INSERT INTO orders(organization_id,location_id,table_id,channel,status)
		VALUES('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000003','salon','entregado'),
		      ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000003','salon','listo');
	`); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, "DROP TABLE pg_temp.orders")
	for _, direction := range []string{"up", "down"} {
		script, readErr := os.ReadFile("../../../migrations/000070_order_delivery_and_completion." + direction + ".sql")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = conn.Exec(ctx, string(script)); err != nil {
			t.Fatalf("migration %s: %v", direction, err)
		}
		if direction == "up" {
			var historicalClosed bool
			if err = conn.QueryRow(ctx, `SELECT completed_at=updated_at FROM orders WHERE status='entregado'`).Scan(&historicalClosed); err != nil || !historicalClosed {
				t.Fatalf("historical delivery must stay closed: closed=%v err=%v", historicalClosed, err)
			}
			if _, err = conn.Exec(ctx, `UPDATE orders SET status='entregado' WHERE status='listo'`); err != nil {
				t.Fatalf("service with pending payment: %v", err)
			}
			if _, err = conn.Exec(ctx, `
				INSERT INTO orders(organization_id,location_id,table_id,channel,status)
				VALUES('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000003','salon','nuevo')
			`); err == nil {
				t.Fatal("unique index must retain a served open table")
			}
		} else {
			var open, closed int
			if err = conn.QueryRow(ctx, `SELECT count(*) FILTER(WHERE status='listo'),count(*) FILTER(WHERE status='entregado') FROM orders`).Scan(&open, &closed); err != nil || open != 1 || closed != 1 {
				t.Fatalf("rollback must preserve open balance and closed history: open=%d closed=%d err=%v", open, closed, err)
			}
		}
	}
}
