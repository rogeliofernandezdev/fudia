package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

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

func TestTableBatchRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	for _, body := range []string{`{`, `{"items":[]}`, `{"items":[{"name":" "}]}`, `{"items":[{"name":"Mesa 1"},{"name":" Mesa 1 "}]}`} {
		req := httptest.NewRequest("POST", "/v1/admin/tables/batch", strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, scope{}))
		rec := httptest.NewRecorder()
		New(nil).createTablesBatch(rec, req)
		if rec.Code != 400 && rec.Code != 409 {
			t.Fatalf("unexpected status %d for %s: %s", rec.Code, body, rec.Body.String())
		}
	}
	items := make([]tableInput, 501)
	body, _ := json.Marshal(tableBatchInput{Items: items})
	req := httptest.NewRequest("POST", "/v1/admin/tables/batch", strings.NewReader(string(body)))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, scope{}))
	rec := httptest.NewRecorder()
	New(nil).createTablesBatch(rec, req)
	if rec.Code != 400 {
		t.Fatalf("oversized batch: %d %s", rec.Code, rec.Body.String())
	}
}
