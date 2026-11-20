package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestOrderListOffersOnlyStatusesWithOrders(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	for _, row := range [][2]string{{"delivery", "nuevo"}, {"delivery", "listo"}, {"salon", "confirmado"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO orders(organization_id,location_id,channel,status,total,created_by) VALUES($1,$2,$3,$4,10,$5)`, s.OrganizationID, s.LocationID, row[0], row[1], s.UserID); err != nil {
			t.Fatal(err)
		}
	}
	read := func(query string) []string {
		t.Helper()
		req := httptest.NewRequest("GET", "/v1/admin/orders?"+query, nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
		rec := httptest.NewRecorder()
		New(pool).listOrders(rec, req)
		if rec.Code != 200 {
			t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
		}
		var body struct {
			StatusOptions []struct{ Value string } `json:"statusOptions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, option := range body.StatusOptions {
			out = append(out, option.Value)
		}
		return out
	}
	if got := read(""); len(got) != 3 || got[0] != "nuevo" || got[1] != "confirmado" || got[2] != "listo" {
		t.Fatalf("all channels: %v", got)
	}
	if got := read("channel=delivery"); len(got) != 2 || got[0] != "nuevo" || got[1] != "listo" {
		t.Fatalf("delivery only: %v", got)
	}
	if got := read("channel=delivery&status=cancelado"); len(got) != 3 || got[2] != "cancelado" {
		t.Fatalf("selected status stays listed: %v", got)
	}
}
