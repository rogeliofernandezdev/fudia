package postgres

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDeliveryLinksWorkWhenPgcryptoIsOutsideSearchPath(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// The extension relocation, isolated schema and fixtures are all rolled back.
	defer func() { _ = tx.Rollback(ctx) }()
	nonce := time.Now().UnixNano()
	hidden := fmt.Sprintf("delivery_extensions_%d", nonce)
	isolated := fmt.Sprintf("delivery_links_%d", nonce)
	if _, err = tx.Exec(ctx, `CREATE SCHEMA `+hidden+`; ALTER EXTENSION pgcrypto SET SCHEMA `+hidden); err != nil {
		t.Skipf("no se puede reubicar pgcrypto en esta base: %v", err)
	}
	if _, err = tx.Exec(ctx, `CREATE SCHEMA `+isolated+`; SET LOCAL search_path TO `+isolated+`,pg_catalog,public`); err != nil {
		t.Fatal(err)
	}
	var resolvable bool
	if err = tx.QueryRow(ctx, `SELECT to_regprocedure('gen_random_bytes(integer)') IS NOT NULL`).Scan(&resolvable); err != nil || resolvable {
		t.Fatalf("pgcrypto should be outside search_path: resolvable=%v err=%v", resolvable, err)
	}
	apply := func(name string) {
		t.Helper()
		script, readErr := os.ReadFile("../../../../../migrations/" + name + ".sql")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, execErr := tx.Exec(ctx, string(script)); execErr != nil {
			t.Fatalf("migration %s without resolvable pgcrypto: %v", name, execErr)
		}
	}
	// Execute the actual initial migration, shadowing the existing table.
	apply("000115_location_delivery_links.up")
	var s scope
	if err = tx.QueryRow(ctx, `INSERT INTO organizations(legal_name,trade_name,tax_id) VALUES($1,$1,$2) RETURNING id`, fmt.Sprintf("Delivery QR %d", nonce), fmt.Sprintf("%011d", nonce%100000000000)).Scan(&s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Principal',$2,'') RETURNING id`, s.OrganizationID, fmt.Sprintf("D%d", nonce%1000000)).Scan(&s.LocationID); err != nil {
		t.Fatal(err)
	}
	repo := New(pool)
	txCtx := database.WithTransaction(ctx, pool, tx)
	validToken := regexp.MustCompile(`^[0-9a-f]{32}$`)
	created, err := repo.RotateDeliveryLink(txCtx, s, s.LocationID)
	if err != nil || !validToken.MatchString(created.Token) || !created.Active {
		t.Fatalf("create link without pgcrypto: %+v err=%v", created, err)
	}
	rotated, err := repo.RotateDeliveryLink(txCtx, s, s.LocationID)
	if err != nil || !validToken.MatchString(rotated.Token) || rotated.Token == created.Token {
		t.Fatalf("rotate link without pgcrypto: %+v err=%v", rotated, err)
	}

	// Simulate the default of an already-applied original 115. PostgreSQL binds
	// that function by OID even when its extension is later moved to a schema.
	if _, err = tx.Exec(ctx, `ALTER TABLE location_delivery_links ALTER COLUMN qr_token SET DEFAULT encode(`+hidden+`.gen_random_bytes(16),'hex')`); err != nil {
		t.Fatal(err)
	}
	apply("000119_delivery_link_native_random.up")
	unchanged, err := repo.DeliveryLink(txCtx, s, s.LocationID)
	if err != nil || unchanged.Token != rotated.Token {
		t.Fatalf("migration must preserve existing QR: %+v err=%v", unchanged, err)
	}
	var defaultSQL string
	if err = tx.QueryRow(ctx, `SELECT pg_get_expr(d.adbin,d.adrelid) FROM pg_attrdef d JOIN pg_attribute a ON a.attrelid=d.adrelid AND a.attnum=d.adnum WHERE d.adrelid='location_delivery_links'::regclass AND a.attname='qr_token'`).Scan(&defaultSQL); err != nil || strings.Contains(defaultSQL, "gen_random_bytes") || !strings.Contains(defaultSQL, "gen_random_uuid") {
		t.Fatalf("default still depends on pgcrypto: %q err=%v", defaultSQL, err)
	}
	apply("000119_delivery_link_native_random.down")
	// Remove the relocated extension entirely: creation and regeneration must
	// keep working even after reverting the corrective migration.
	if _, err = tx.Exec(ctx, `DROP EXTENSION pgcrypto; DELETE FROM location_delivery_links`); err != nil {
		t.Fatal(err)
	}
	created, err = repo.RotateDeliveryLink(txCtx, s, s.LocationID)
	if err != nil || !validToken.MatchString(created.Token) {
		t.Fatalf("create after correction without extension: %+v err=%v", created, err)
	}
	rotated, err = repo.RotateDeliveryLink(txCtx, s, s.LocationID)
	if err != nil || !validToken.MatchString(rotated.Token) || rotated.Token == created.Token {
		t.Fatalf("rotate after correction without extension: %+v err=%v", rotated, err)
	}
}
