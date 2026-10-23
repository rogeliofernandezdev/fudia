package httpapi

import (
	"context"
	"os"
	"testing"
)

func TestKardexOrderNotesMigrationReplacesOnlyTheRawOrderID(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	// A temporary shadow table exercises both scripts without touching the
	// migrated Kárdex or other integration fixtures.
	if _, err = conn.Exec(ctx, `
		CREATE TEMP TABLE stock_movements (
		  id text PRIMARY KEY,
		  movement_type text NOT NULL,
		  source_type text NOT NULL,
		  source_id uuid NOT NULL,
		  note text NOT NULL DEFAULT ''
		);
		INSERT INTO stock_movements(id,movement_type,source_type,source_id,note)
		VALUES('sale','sale','order','00000000-0000-0000-0000-000000000001','Pedido 00000000-0000-0000-0000-000000000001'),
		      ('reversal','sale_reversal','order','00000000-0000-0000-0000-000000000002','Pedido 00000000-0000-0000-0000-000000000002'),
		      ('adjustment','sale_adjustment','order','00000000-0000-0000-0000-000000000003','Pedido 00000000-0000-0000-0000-000000000003'),
		      ('other-order','sale','order','00000000-0000-0000-0000-000000000004','Pedido 00000000-0000-0000-0000-000000000009'),
		      ('recipe','recipe_consumption','order','00000000-0000-0000-0000-000000000005','Consumo automático por receta'),
		      ('entry','entry','inventory_entry','00000000-0000-0000-0000-000000000006','Pedido 00000000-0000-0000-0000-000000000006');
	`); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, "DROP TABLE pg_temp.stock_movements")
	want := map[string]map[string]string{
		"up": {
			"sale":        "Salida automática por venta",
			"reversal":    "Reversa de venta",
			"adjustment":  "Ajuste por edición del pedido",
			"other-order": "Pedido 00000000-0000-0000-0000-000000000009",
			"recipe":      "Consumo automático por receta",
			"entry":       "Pedido 00000000-0000-0000-0000-000000000006",
		},
		"down": {
			"sale":        "Pedido 00000000-0000-0000-0000-000000000001",
			"reversal":    "Pedido 00000000-0000-0000-0000-000000000002",
			"adjustment":  "Pedido 00000000-0000-0000-0000-000000000003",
			"other-order": "Pedido 00000000-0000-0000-0000-000000000009",
			"recipe":      "Consumo automático por receta",
			"entry":       "Pedido 00000000-0000-0000-0000-000000000006",
		},
	}
	for _, direction := range []string{"up", "down"} {
		script, readErr := os.ReadFile("../../../migrations/000086_kardex_order_notes." + direction + ".sql")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = conn.Exec(ctx, string(script)); err != nil {
			t.Fatalf("migration %s: %v", direction, err)
		}
		for id, note := range want[direction] {
			var got string
			if err = conn.QueryRow(ctx, `SELECT note FROM stock_movements WHERE id=$1`, id).Scan(&got); err != nil || got != note {
				t.Fatalf("%s %s: note=%q want %q err=%v", direction, id, got, note, err)
			}
		}
	}
}
