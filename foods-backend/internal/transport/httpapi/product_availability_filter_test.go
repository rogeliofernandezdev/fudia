package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestAvailabilityProductIDsFilter(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	other := seedInventoryScope(t, pool)
	first := seedPortionAvailability(t, pool, s, 5, 2)
	_ = seedPortionAvailability(t, pool, s, 20, 1)
	foreign := seedPortionAvailability(t, pool, other, 10, 0)
	api := New(pool)
	read := func() struct {
		Items []struct {
			ProductID string   `json:"productId"`
			Status    string   `json:"status"`
			Remaining *float64 `json:"remaining"`
		}
	} { t.Helper(); request := httptest.NewRequest("GET", "/v1/admin/product-availability?pageSize=100&productIds="+first+","+foreign, nil); request = request.WithContext(context.WithValue(request.Context(), scopeKey{}, s)); response := httptest.NewRecorder(); api.listProductAvailability(response, request); if response.Code != 200 {
		t.Fatalf("availability: %d %s", response.Code, response.Body.String())
	}; var out struct {
		Items []struct {
			ProductID string   `json:"productId"`
			Status    string   `json:"status"`
			Remaining *float64 `json:"remaining"`
		}
	}; if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}; return out }
	out := read()
	if len(out.Items) != 1 || out.Items[0].ProductID != first || out.Items[0].Remaining == nil || *out.Items[0].Remaining != 3 {
		t.Fatalf("scoped batch returned unexpected data: %+v", out)
	}
	changed := updateAvailabilityRequest(t, api, s, first, `{"status":"sold_out","reasonCode":"sold_out"}`)
	if changed.Code != 204 {
		t.Fatalf("mark sold out: %d %s", changed.Code, changed.Body.String())
	}
	out = read()
	if len(out.Items) != 1 || out.Items[0].Status != "sold_out" {
		t.Fatalf("batch did not refresh sold-out status: %+v", out)
	}
}
