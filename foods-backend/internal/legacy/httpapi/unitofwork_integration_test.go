package httpapi

import (
	"context"
	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	identitypg "github.com/foods-platform/foods-backend/internal/modules/identity/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"testing"
)

// Two independent adapters must see the same uncommitted writes and roll back together.
func TestUnitOfWorkSharesTransactionAcrossRepositories(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	repo := identitypg.NewAdministration(pool)
	audit := auditpg.New(pool)
	work := database.NewUnitOfWork(pool)
	var before string
	if err := pool.QueryRow(ctx, `SELECT full_name FROM users WHERE id=$1`, s.UserID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	for _, commit := range []bool{false, true} {
		tx, err := work.Begin(ctx, unitofwork.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = work.Begin(tx.Context(), unitofwork.Options{}); err == nil {
			t.Fatal("nested transaction silently opened")
		}
		if err = repo.UpdateProfile(tx.Context(), s.UserID, "Shared transaction", "hash"); err != nil {
			t.Fatal(err)
		}
		profile, err := repo.Profile(tx.Context(), s)
		if err != nil || profile.FullName != "Shared transaction" {
			t.Fatalf("transactional read: %+v %v", profile, err)
		}
		if err = audit.Record(tx.Context(), s, "uow.test", "user", s.UserID); err != nil {
			t.Fatal(err)
		}
		var outside string
		if err = pool.QueryRow(ctx, `SELECT full_name FROM users WHERE id=$1`, s.UserID).Scan(&outside); err != nil || outside != before {
			t.Fatalf("uncommitted state escaped: %q %v", outside, err)
		}
		if commit {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		var count int
		if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND action='uow.test'`, s.OrganizationID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 0
		expected := before
		if commit {
			want = 1
			expected = "Shared transaction"
		}
		if count != want {
			t.Fatalf("audit count=%d want %d", count, want)
		}
		if err = pool.QueryRow(ctx, `SELECT full_name FROM users WHERE id=$1`, s.UserID).Scan(&outside); err != nil || outside != expected {
			t.Fatalf("profile after completion: %q %v", outside, err)
		}
	}
}
