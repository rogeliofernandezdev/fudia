package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

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
