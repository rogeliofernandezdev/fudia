package http

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAvailabilityRejectsInvalidProductIDFilters(t *testing.T) {
	id := "4f449f72-9469-4c56-a2bc-ebdd923f73ec"
	for _, query := range []string{"productIds=", "productIds=invalid", "productIds=" + id + "&productIds=" + id, "productIds=" + strings.Repeat(id+",", 100) + id} {
		recorder := httptest.NewRecorder()
		NewCatalog(nil).ListProductAvailability(recorder, httptest.NewRequest("GET", "/availability?"+query, nil))
		if recorder.Code != 400 {
			t.Fatalf("expected invalid filter to return 400, got %d", recorder.Code)
		}
	}
}
