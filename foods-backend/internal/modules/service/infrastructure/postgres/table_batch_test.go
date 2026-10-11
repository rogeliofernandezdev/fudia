package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/modules/service/domain"
	"github.com/jackc/pgx/v5"
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
