package httpapi

import (
	"context"
	"os"
	"testing"
)

func TestKardexNotesMigrationRemovesInternalCodes(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	// A temporary shadow table exercises the script without touching the migrated Kárdex.
	if _, err = conn.Exec(ctx, `
		CREATE TEMP TABLE stock_movements (
		  id text PRIMARY KEY,
		  movement_type text NOT NULL,
		  source_type text NOT NULL,
		  note text NOT NULL DEFAULT ''
		);
		INSERT INTO stock_movements(id,movement_type,source_type,note)
		VALUES('receipt','entry','purchase_receipt','Recepción REC-F61D02B0 de OC-C528D2EB'),
		      ('receipt-notes','entry','purchase_receipt','Recepción REC-F61D02B0 de OC-C528D2EB · llegó en dos cajas'),
		      ('return','supplier_return','purchase_return','DEV-1A2B3C4D · mercadería defectuosa'),
		      ('out','transfer_out','inventory_transfer','TRF-0A1B2C3D → Local Miraflores'),
		      ('in','transfer_in','inventory_transfer','TRF-0A1B2C3D desde transferencia'),
		      ('typed','entry','purchase_receipt','Recepción sin factura');
	`); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, "DROP TABLE pg_temp.stock_movements")
	script, err := os.ReadFile("../../../migrations/000088_kardex_notes_without_codes.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Exec(ctx, string(script)); err != nil {
		t.Fatalf("migration up: %v", err)
	}
	for id, want := range map[string]string{
		"receipt":       "",
		"receipt-notes": "llegó en dos cajas",
		"return":        "mercadería defectuosa",
		"out":           "",
		"in":            "",
		"typed":         "Recepción sin factura",
	} {
		var got string
		if err = conn.QueryRow(ctx, `SELECT note FROM stock_movements WHERE id=$1`, id).Scan(&got); err != nil || got != want {
			t.Fatalf("%s: note=%q want %q err=%v", id, got, want, err)
		}
	}
}
