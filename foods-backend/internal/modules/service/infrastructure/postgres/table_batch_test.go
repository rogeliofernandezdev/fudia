package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/modules/service/application"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"strings"
	"testing"
)

type tableBatchItem = domain.TableBatchItem
type batchQueryCounter struct{ calls int }

func (q *batchQueryCounter) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	q.calls++
	if query != tableBatchSQL || len(args) != 4 {
		return batchTestRow{err: errors.New("unexpected batch query")}
	}
	return batchTestRow{}
}

type batchTestRow struct{ err error }

func (row batchTestRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*bool) = false
	*dest[1].(*json.RawMessage) = json.RawMessage(`{"items":[]}`)
	return nil
}

func TestTableBatchUsesOneDatabaseRoundTrip(t *testing.T) {
	for _, size := range []int{1, 10, 50, 500} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			items := make([]tableBatchItem, size)
			for i := range items {
				items[i] = tableBatchItem{Name: fmt.Sprintf("Mesa %d", i), Seats: 2, Zone: "Principal", QrEnabled: true}
			}
			payload, _ := json.Marshal(items)
			q := &batchQueryCounter{}
			if _, invalid, err := persistTableBatch(context.Background(), q, scope{}, payload); err != nil || invalid || q.calls != 1 {
				t.Fatalf("expected one round trip for %d tables: calls=%d invalid=%v err=%v", size, q.calls, invalid, err)
			}
		})
	}
}

func TestTableQRTokensDoNotDependOnPgcrypto(t *testing.T) {
	source, err := os.ReadFile("floor.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "gen_random_bytes(") {
		t.Fatal("QR tokens must not call pgcrypto.gen_random_bytes; restored databases may keep pgcrypto outside the search_path")
	}
	if !strings.Contains(tableBatchSQL, qrTokenSQL) {
		t.Fatal("batch must generate QR tokens with qrTokenSQL")
	}
}

func TestTableBatchErrorSeparatesRejectedFromUncertainWrites(t *testing.T) {
	rejected := batchError(&pgconn.PgError{Code: "42883", Message: "function gen_random_bytes(integer) does not exist"})
	if !errors.Is(rejected, application.ErrNotSaved) {
		t.Fatalf("a statement rejected by PostgreSQL rolled back the batch: %v", rejected)
	}
	var pg *pgconn.PgError
	if !errors.As(rejected, &pg) || pg.Code != "42883" {
		t.Fatalf("the PostgreSQL cause must remain available for logs: %v", rejected)
	}
	if err := batchError(&pgconn.PgError{Code: "23505"}); !errors.Is(err, application.ErrConflict) || errors.Is(err, application.ErrNotSaved) {
		t.Fatalf("duplicates stay as conflicts: %v", err)
	}
	for _, uncertain := range []error{context.Canceled, context.DeadlineExceeded, errors.New("unexpected EOF")} {
		if err := batchError(uncertain); errors.Is(err, application.ErrNotSaved) {
			t.Fatalf("%v may have committed and must stay unconfirmed", uncertain)
		}
	}
}
