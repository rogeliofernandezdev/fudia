package postgres

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Reproduces a restored database whose pgcrypto lives outside the search_path:
// gen_random_uuid() still works, but gen_random_bytes() cannot be resolved.
func TestTableWritesWorkWhenPgcryptoIsOutsideSearchPath(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Everything below, including the moved extension, is rolled back.
	defer func() { _ = tx.Rollback(ctx) }()
	nonce := time.Now().UnixNano()
	hidden := fmt.Sprintf("restored_extensions_%d", nonce)
	if _, err = tx.Exec(ctx, `CREATE SCHEMA `+hidden+`; ALTER EXTENSION pgcrypto SET SCHEMA `+hidden); err != nil {
		t.Skipf("no se puede reubicar pgcrypto en esta base: %v", err)
	}
	var resolvable bool
	if err = tx.QueryRow(ctx, `SELECT to_regprocedure('gen_random_bytes(integer)') IS NOT NULL`).Scan(&resolvable); err != nil || resolvable {
		t.Fatalf("pgcrypto should be outside the search_path: resolvable=%v err=%v", resolvable, err)
	}
	var s scope
	if err = tx.QueryRow(ctx, `INSERT INTO organizations(legal_name,trade_name,tax_id) VALUES($1,$1,$2) RETURNING id`, fmt.Sprintf("QR Test %d", nonce), fmt.Sprintf("%011d", nonce%100000000000)).Scan(&s.OrganizationID); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address) VALUES($1,'Principal',$2,'') RETURNING id`, s.OrganizationID, fmt.Sprintf("Q%d", nonce%1000000)).Scan(&s.LocationID); err != nil {
		t.Fatalf("seed location: %v", err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,'QR Test','test') RETURNING id`, s.OrganizationID, fmt.Sprintf("qr-%d@example.test", nonce)).Scan(&s.UserID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	repo := NewFloor(pool)
	txCtx := database.WithTransaction(ctx, pool, tx)
	token := regexp.MustCompile(`^[0-9a-f]{32}$`)
	out, invalid, err := repo.Batch(txCtx, s, []domain.TableBatchItem{{Name: "Mesa 1", Seats: 2, QrEnabled: true}, {Name: "Mesa 2", Seats: 4, QrEnabled: true}})
	if err != nil || invalid || len(out.Items) != 2 {
		t.Fatalf("batch without resolvable pgcrypto: items=%d invalid=%v err=%v", len(out.Items), invalid, err)
	}
	if !token.MatchString(out.Items[0].QrToken) || out.Items[0].QrToken == out.Items[1].QrToken {
		t.Fatalf("unexpected QR tokens: %q %q", out.Items[0].QrToken, out.Items[1].QrToken)
	}
	single, err := repo.CreateTable(txCtx, s, "Mesa 3", 2, "", true)
	if err != nil || !token.MatchString(single.QrToken) {
		t.Fatalf("single table without resolvable pgcrypto: %+v err=%v", single, err)
	}
	regenerated, err := repo.RegenerateQR(txCtx, s, single.ID)
	if err != nil || !token.MatchString(regenerated.QrToken) || regenerated.QrToken == single.QrToken {
		t.Fatalf("regenerated QR without resolvable pgcrypto: %+v err=%v", regenerated, err)
	}
}
