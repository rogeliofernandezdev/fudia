package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPreparationPermissionsAreIndependentByStationAndLocation(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = seedOrganizationRoles(ctx, tx, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"bartender", "cook", "administrator", "location_manager", "shift_supervisor"} {
		if _, err = pool.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, s.UserID); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) SELECT $1,id,$2 FROM roles WHERE organization_id=$3 AND system_key=$4`, s.UserID, s.LocationID, s.OrganizationID, key); err != nil {
			t.Fatal(err)
		}
		for _, station := range []string{"kitchen", "bar"} {
			id := "00000000-0000-0000-0000-000000000001"
			if station == "bar" {
				id += "~bar"
			}
			req := httptest.NewRequest("PATCH", "/", nil)
			req.SetPathValue("id", id)
			req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
			rec := httptest.NewRecorder()
			api.requirePreparationPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(rec, req)
			want := 204
			if (key == "bartender" && station == "kitchen") || (key != "bartender" && key != "administrator" && station == "bar") {
				want = 403
			}
			if rec.Code != want {
				t.Fatalf("%s / %s: got %d want %d: %s", key, station, rec.Code, want, rec.Body.String())
			}
		}
	}
	foreign := s
	foreign.LocationID = "00000000-0000-0000-0000-000000000002"
	req := httptest.NewRequest("PATCH", "/", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001~bar")
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, foreign))
	rec := httptest.NewRecorder()
	api.requirePreparationPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("station permission crossed location") })).ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatal(rec.Body.String())
	}
}
