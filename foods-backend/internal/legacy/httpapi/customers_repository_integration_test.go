package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	customer "github.com/foods-platform/foods-backend/internal/modules/customers/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCustomerRepositoryScopeAndAddressRollback(t *testing.T) {
	pool := integrationPool(t)
	s, other := seedInventoryScope(t, pool), seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE organization_id=ANY($1::uuid[])`, []string{s.OrganizationID, other.OrganizationID})
	})
	call := func(scope scope, method, id, body string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/v1/admin/customers", strings.NewReader(body))
		r.SetPathValue("id", id)
		r = r.WithContext(context.WithValue(r.Context(), scopeKey{}, scope))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	created := call(s, "POST", "", `{"displayName":" Ana ","addresses":[{"address":"Calle 1","default":true}]}`, api.createCustomer)
	var c customer.Customer
	if created.Code != 201 || json.Unmarshal(created.Body.Bytes(), &c) != nil || c.ID == "" || c.DisplayName != "Ana" {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	read := call(s, "GET", c.ID, "", api.getCustomer)
	if read.Code != 200 || !strings.Contains(read.Body.String(), "Calle 1") {
		t.Fatalf("read addresses: %s", read.Body.String())
	}
	if got := call(other, "GET", c.ID, "", api.getCustomer); got.Code != 404 {
		t.Fatalf("cross tenant read: %d", got.Code)
	}
	if got := call(other, "PATCH", c.ID, `{"displayName":"Intruder"}`, api.updateCustomer); got.Code != 404 {
		t.Fatalf("cross tenant update: %d", got.Code)
	}
	if got := call(other, "PATCH", c.ID, `{"active":false}`, api.updateCustomerStatus); got.Code != 404 {
		t.Fatalf("cross tenant status: %d", got.Code)
	}
	failed := call(s, "PATCH", c.ID, `{"displayName":"Changed","addresses":[{"address":"Invalid","countryCode":"INVALID"}]}`, api.updateCustomer)
	if failed.Code != 400 {
		t.Fatalf("expected rejected address, got %d %s", failed.Code, failed.Body.String())
	}
	read = call(s, "GET", c.ID, "", api.getCustomer)
	if read.Code != 200 || !strings.Contains(read.Body.String(), "Calle 1") || !strings.Contains(read.Body.String(), `"displayName":"Ana"`) {
		t.Fatalf("update did not roll back aggregate: %s", read.Body.String())
	}
	failed = call(s, "POST", "", `{"displayName":"Rejected","addresses":[{"address":"Invalid","countryCode":"INVALID"}]}`, api.createCustomer)
	if failed.Code != 400 {
		t.Fatalf("create bad address: %d", failed.Code)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM customers WHERE organization_id=$1`, s.OrganizationID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("partial customer persisted: %d %v", count, err)
	}
	if got := call(other, "GET", "", "", api.listCustomers); got.Code != 200 || strings.Contains(got.Body.String(), c.ID) {
		t.Fatalf("cross tenant list: %s", got.Body.String())
	}
	updated := call(s, "PATCH", c.ID, `{"displayName":"Ana updated","addresses":[{"address":"Calle 2"}]}`, api.updateCustomer)
	if updated.Code != 200 {
		t.Fatalf("update: %s", updated.Body.String())
	}
	if got := call(s, "PATCH", c.ID, `{"active":false}`, api.updateCustomerStatus); got.Code != 204 {
		t.Fatalf("status: %d", got.Code)
	}
}
func TestCustomerRoutesRetainAuthenticationAndPermissions(t *testing.T) {
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	var role string
	if err := pool.QueryRow(ctx, `INSERT INTO roles(organization_id,name,permissions) VALUES($1,'Customer reader',ARRAY['customers.read']) RETURNING id`, s.OrganizationID).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, s.UserID, role, s.LocationID); err != nil {
		t.Fatal(err)
	}
	token := "customer-reader-" + s.UserID
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`, hash[:], s.UserID, s.OrganizationID, s.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	routes := New(pool).Routes()
	for _, tc := range []struct {
		method string
		auth   bool
		want   int
	}{{"GET", false, 401}, {"GET", true, 200}, {"POST", true, 403}} {
		r := httptest.NewRequest(tc.method, "/v1/admin/customers", strings.NewReader(`{"displayName":"Must not save"}`))
		if tc.auth {
			r.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		}
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s auth=%v: got %d want %d body %s", tc.method, tc.auth, w.Code, tc.want, w.Body.String())
		}
	}
}
