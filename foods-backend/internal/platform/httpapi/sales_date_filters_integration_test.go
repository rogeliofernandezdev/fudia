package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestSalesDateFiltersUseLocalDaysAndPagination(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='America/Lima' WHERE id=$1 AND organization_id=$2`, s.LocationID, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	// Lima 7 October runs from 05:00Z inclusive until 05:00Z on 8 October exclusive.
	for _, item := range []struct{ code, at string }{
		{"BEFORE", "2026-10-07T04:59:59Z"},
		{"START", "2026-10-07T05:00:00Z"},
		{"MIDDLE", "2026-10-07T17:00:00Z"},
		{"END", "2026-10-08T04:59:59.999999Z"},
		{"AFTER", "2026-10-08T05:00:00Z"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO orders(organization_id,location_id,code,channel,status,total,created_by,created_at)
			VALUES($1,$2,$3,'mostrador','listo',0,$4,$5::timestamptz)`, s.OrganizationID, s.LocationID, item.code, s.UserID, item.at); err != nil {
			t.Fatal(err)
		}
	}
	type response struct {
		Items []posOrderSummary `json:"items"`
		Total int               `json:"total"`
	}
	list := func(query string) response {
		t.Helper()
		r := httptest.NewRequest("GET", "/v1/admin/pos/orders?paymentStatus=paid&"+query, nil)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, s))
		w := httptest.NewRecorder()
		api.listPOSOrders(w, r)
		if w.Code != 200 {
			t.Fatalf("list: %d %s", w.Code, w.Body.String())
		}
		var out response
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	for _, tc := range []struct {
		query string
		codes []string
	}{
		{"from=2026-10-07&to=2026-10-07", []string{"END", "MIDDLE", "START"}},
		{"from=2026-10-07", []string{"AFTER", "END", "MIDDLE", "START"}},
		{"to=2026-10-07", []string{"END", "MIDDLE", "START", "BEFORE"}},
		{"", []string{"AFTER", "END", "MIDDLE", "START", "BEFORE"}},
		{"from=2026-10-09&to=2026-10-09", []string{}},
		{"from=2026-10-07&to=2026-10-07&q=MIDDLE", []string{"MIDDLE"}},
		{"from=" + url.QueryEscape(" 2026-10-07 ") + "&to=2026-10-07", []string{"END", "MIDDLE", "START"}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			out := list(tc.query)
			codes := []string{}
			for _, item := range out.Items {
				codes = append(codes, item.Code)
			}
			if out.Total != len(tc.codes) || !reflect.DeepEqual(codes, tc.codes) {
				t.Fatalf("want %v, got %#v", tc.codes, out)
			}
		})
	}
	out := list("from=2026-10-07&to=2026-10-07&pageSize=1&page=2")
	if out.Total != 3 || len(out.Items) != 1 || out.Items[0].Code != "MIDDLE" {
		t.Fatalf("filtered pagination: %#v", out)
	}
	// The bounds come from the location, not a fixed Peru/UTC offset.
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='UTC' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	out = list("from=2026-10-07&to=2026-10-07")
	if out.Total != 3 || out.Items[0].Code != "MIDDLE" || out.Items[2].Code != "BEFORE" {
		t.Fatalf("UTC bounds: %#v", out)
	}
	// A daylight-saving day has 23 hours: do not add 24 hours to a UTC bound.
	if _, err := pool.Exec(ctx, `UPDATE locations SET timezone='America/New_York' WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ code, at string }{
		{"BEFORE", "2026-03-08T04:59:59Z"},
		{"START", "2026-03-08T05:00:00Z"},
		{"MIDDLE", "2026-03-08T17:00:00Z"},
		{"END", "2026-03-09T03:59:59.999999Z"},
		{"AFTER", "2026-03-09T04:00:00Z"},
	} {
		if _, err := pool.Exec(ctx, `UPDATE orders SET created_at=$1::timestamptz WHERE code=$2 AND organization_id=$3`, item.at, item.code, s.OrganizationID); err != nil {
			t.Fatal(err)
		}
	}
	out = list("from=2026-03-08&to=2026-03-08")
	if out.Total != 3 || out.Items[0].Code != "END" || out.Items[2].Code != "START" {
		t.Fatalf("daylight-saving bounds: %#v", out)
	}
}
