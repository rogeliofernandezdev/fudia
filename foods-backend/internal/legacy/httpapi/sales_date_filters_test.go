package httpapi

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestSalesListRejectsInvalidDateFiltersBeforeQuery(t *testing.T) {
	api := New(nil)
	for _, query := range []string{
		"from=2026-02-30", "to=not-a-date", "from=2026-10-07&to=2026-10-01",
		"from=0000-01-01", "to=07-10-2026",
	} {
		t.Run(query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/v1/admin/pos/orders?"+query, nil)
			r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, scope{}))
			w := httptest.NewRecorder()
			api.listPOSOrders(w, r)
			if w.Code != 400 {
				t.Fatalf("expected invalid range 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
