package httpapi

import (
	"context"
	"os"
	"testing"
)

func TestAutoCompletionMigrationOnlyClosesDeliveredNetPaidAccounts(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `
		CREATE TEMP TABLE orders(id int PRIMARY KEY,organization_id int,location_id int,channel text,status text,total numeric,completed_at timestamptz,updated_at timestamptz DEFAULT now()) ON COMMIT DROP;
		CREATE TEMP TABLE payments(id int PRIMARY KEY,order_id int,organization_id int,location_id int,amount numeric) ON COMMIT DROP;
		CREATE TEMP TABLE payment_refunds(payment_id int,organization_id int,location_id int,amount numeric) ON COMMIT DROP;
		INSERT INTO orders(id,organization_id,location_id,channel,status,total) VALUES
		  (1,1,1,'salon','entregado',25), (2,1,1,'salon','entregado',25),
		  (3,1,1,'salon','listo',25), (4,1,1,'salon','entregado',25),
		  (5,1,1,'salon','entregado',25), (6,1,1,'mostrador','entregado',25),
		  (7,1,1,'salon','entregado',25), (8,1,1,'salon','entregado',0),
		  (9,1,1,'salon','entregado',25);
		UPDATE orders SET completed_at='2026-01-01Z' WHERE id=7;
		INSERT INTO payments VALUES (1,1,1,1,25),(2,2,1,1,10),(3,3,1,1,25),(4,4,1,1,25),(5,5,1,2,25),(6,6,1,1,25),(9,9,2,1,25);
		INSERT INTO payment_refunds VALUES(4,1,1,5);
	`); err != nil {
		t.Fatal(err)
	}
	for _, direction := range []string{"up", "up", "down"} {
		script, err := os.ReadFile("../../../migrations/000071_auto_complete_paid_salon_orders." + direction + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(ctx, string(script)); err != nil {
			t.Fatal(err)
		}
		var correct bool
		if err = tx.QueryRow(ctx, `SELECT bool_and((completed_at IS NOT NULL)=(id IN (1,7,8))) FROM orders`).Scan(&correct); err != nil || !correct {
			t.Fatalf("%s must preserve unpaid, undelivered, refunded and foreign-scope balances: correct=%v err=%v", direction, correct, err)
		}
		if err = tx.QueryRow(ctx, `SELECT completed_at='2026-01-01Z' FROM orders WHERE id=7`).Scan(&correct); err != nil || !correct {
			t.Fatalf("historical closure changed: %v", err)
		}
	}
}
