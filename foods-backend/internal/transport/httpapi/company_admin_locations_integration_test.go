package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	billingpostgres "github.com/foods-platform/foods-backend/internal/modules/billing/infrastructure/postgres"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	identitypostgres "github.com/foods-platform/foods-backend/internal/modules/identity/infrastructure/postgres"
)

func seedCompanyLocation(t *testing.T, api *integrationAPI, s scope, name string) string {
	t.Helper()
	var id string
	err := api.db.QueryRow(context.Background(), `
		INSERT INTO locations(organization_id,name,code,fiscal_profile_id)
		SELECT $1,$2,$3,id FROM organization_fiscal_profiles
		WHERE organization_id=$1 AND active ORDER BY is_default DESC LIMIT 1
		RETURNING id`, s.OrganizationID, name, fmt.Sprintf("test-%d", time.Now().UnixNano())).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func companySession(t *testing.T, api *integrationAPI, s scope) func(string, string, any) *httptest.ResponseRecorder {
	t.Helper()
	ctx := context.Background()
	_, err := api.db.Exec(ctx, `UPDATE locations SET fiscal_profile_id=(SELECT id FROM organization_fiscal_profiles WHERE organization_id=$1 AND active ORDER BY is_default DESC LIMIT 1) WHERE id=$2`, s.OrganizationID, s.LocationID)
	if err != nil {
		t.Fatal(err)
	}
	token := fmt.Sprintf("company-locations-%d", time.Now().UnixNano())
	if err := identitypostgres.New(api.db).CreateSession(ctx, identitydomain.Session{Hash: sha256Sum(token), UserID: s.UserID, OrganizationID: s.OrganizationID, LocationID: s.LocationID, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = api.db.Exec(context.Background(), `DELETE FROM sessions WHERE token_hash=$1`, sha256Sum(token))
	})
	routes := api.Routes()
	return func(method, path string, body any) *httptest.ResponseRecorder {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.AddCookie(&http.Cookie{Name: "foods_session", Value: token})
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}
}

func TestCompanyAdministratorUsesAllOwnLocations(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	ctx := context.Background()
	_, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',name='Administrador de empresa',permissions=$2,menu_access=$3 WHERE id=$1`, role, identitydomain.CompanyAdministratorPermissions, identitydomain.CompanyAdministratorMenuAccess)
	if err != nil {
		t.Fatal(err)
	}
	second := seedCompanyLocation(t, api, s, "Segundo local")
	call := companySession(t, api, s)
	list := func(want int) {
		t.Helper()
		rec := call("GET", "/v1/admin/locations/available", nil)
		var got struct {
			Items []locationSummary `json:"items"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &got) != nil || len(got.Items) != want {
			t.Fatalf("available locals: status=%d body=%s; want %d", rec.Code, rec.Body.String(), want)
		}
	}
	list(2)
	third := seedCompanyLocation(t, api, s, "Tercer local")
	list(3) // A newly created local needs no additional persisted assignment.
	var assignments int
	if err := api.db.QueryRow(ctx, `SELECT count(*) FROM user_roles WHERE user_id=$1`, s.UserID).Scan(&assignments); err != nil || assignments != 1 {
		t.Fatalf("assignments unexpectedly copied: %d, %v", assignments, err)
	}
	for _, target := range []string{second, third} {
		rec := call("POST", "/v1/admin/switch-context", switchContextInput{OrganizationID: s.OrganizationID, LocationID: target})
		if rec.Code != 200 {
			t.Fatalf("switch own local: %d %s", rec.Code, rec.Body.String())
		}
		rec = call("GET", "/v1/admin/context", nil)
		var got struct {
			User struct {
				PlatformAdmin bool `json:"platformAdmin"`
			} `json:"user"`
			Location struct {
				ID string `json:"id"`
			} `json:"location"`
			Permissions []string `json:"permissions"`
			MenuAccess  []string `json:"menuAccess"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &got) != nil || got.Location.ID != target || got.User.PlatformAdmin || !containsString(got.Permissions, "orders.manage") || !containsString(got.MenuAccess, "pedidos") {
			t.Fatalf("context lost tenant access: %d %s", rec.Code, rec.Body.String())
		}
		profile := call("GET", "/v1/admin/me", nil)
		if profile.Code != 200 || !bytes.Contains(profile.Body.Bytes(), []byte("Administrador de empresa")) {
			t.Fatalf("profile after switch: %d %s", profile.Code, profile.Body.String())
		}
		targetScope := s
		targetScope.LocationID = target
		allowed, err := identitypostgres.New(api.db).HasPermission(ctx, targetScope, "orders.manage")
		if err != nil || !allowed {
			t.Fatalf("fallback authorization: %v %v", allowed, err)
		}
		visible, err := identitypostgres.NewAdministration(api.db).CashExpectedVisible(ctx, targetScope)
		if err != nil || !visible {
			t.Fatalf("context cash permission: %v %v", visible, err)
		}
		cash := billingpostgres.NewPayments(api.db)
		visible, err = cash.CanSeeExpected(ctx, targetScope)
		if err != nil || !visible {
			t.Fatalf("cash permission: %v %v", visible, err)
		}
		if _, err := cash.AssignableUser(ctx, targetScope, billingdomain.ShiftUserInput{UserID: s.UserID}); err == nil {
			t.Fatal("administrator must not be assignable as a collaborator")
		}
		options, err := cash.UserOptions(ctx, targetScope)
		found := false
		for _, option := range options {
			found = found || option.ID == s.UserID
		}
		if err != nil || found {
			t.Fatalf("administrator included in collaborator options: %v", err)
		}
	}
	other := seedInventoryScope(t, api.db)
	foreign := seedCompanyLocation(t, api, other, "Local ajeno")
	rec := call("POST", "/v1/admin/switch-context", switchContextInput{OrganizationID: other.OrganizationID, LocationID: foreign})
	if rec.Code != 403 {
		t.Fatalf("company admin crossed tenant: %d %s", rec.Code, rec.Body.String())
	}
	if ok, err := identitypostgres.NewAdministration(api.db).HasLocationRole(ctx, s.UserID, foreign); err != nil || ok {
		t.Fatalf("foreign membership allowed: %v %v", ok, err)
	}
	foreignHash := sha256Sum(fmt.Sprintf("invalid-company-context-%d", time.Now().UnixNano()))
	if err := identitypostgres.New(api.db).CreateSession(ctx, identitydomain.Session{Hash: foreignHash, UserID: s.UserID, OrganizationID: s.OrganizationID, LocationID: foreign, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = api.db.Exec(context.Background(), `DELETE FROM sessions WHERE token_hash=$1`, foreignHash)
	})
	if _, _, err := identitypostgres.New(api.db).SessionAccess(ctx, foreignHash); err == nil {
		t.Fatal("authentication accepted a local belonging to another tenant")
	}
	if _, err := api.db.Exec(ctx, `UPDATE locations SET active=false WHERE id=$1`, third); err != nil {
		t.Fatal(err)
	}
	list(2)
	rec = call("POST", "/v1/admin/switch-context", switchContextInput{OrganizationID: s.OrganizationID, LocationID: third})
	if rec.Code != 404 {
		t.Fatalf("inactive local allowed: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := api.db.Exec(ctx, `UPDATE locations SET active=false WHERE id=$1`, s.LocationID); err != nil {
		t.Fatal(err)
	}
	first, err := identitypostgres.New(api.db).FirstLocation(ctx, identitydomain.Candidate{UserID: s.UserID, OrganizationID: s.OrganizationID})
	if err != nil || first != second {
		t.Fatalf("admin cannot sign in after original local deactivation: %s %v", first, err)
	}
}

func TestCompanyLocationAccessDoesNotExpandOtherRoles(t *testing.T) {
	api, s, role := seedIdentityAdministrator(t)
	ctx := context.Background()
	if _, err := api.db.Exec(ctx, `UPDATE roles SET name='Administrador de empresa',permissions=ARRAY['users.manage','orders.manage'],menu_access=ARRAY['usuarios','pedidos'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	second := seedCompanyLocation(t, api, s, "Sin asignación")
	call := companySession(t, api, s)
	assertRestricted := func(want int) {
		t.Helper()
		rec := call("GET", "/v1/admin/locations/available", nil)
		var got struct {
			Items []locationSummary `json:"items"`
		}
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &got) != nil || len(got.Items) != want {
			t.Fatalf("unexpected locations: %d %s", rec.Code, rec.Body.String())
		}
		rec = call("POST", "/v1/admin/switch-context", switchContextInput{OrganizationID: s.OrganizationID, LocationID: second})
		if rec.Code != 403 {
			t.Fatalf("non administrator expanded by name/permissions: %d %s", rec.Code, rec.Body.String())
		}
	}
	assertRestricted(1)
	if _, err := api.db.Exec(ctx, `UPDATE roles SET system_key='administrator',active=false WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	assertRestricted(0)
	if _, err := api.db.Exec(ctx, `UPDATE roles SET active=true WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, s.UserID); err != nil {
		t.Fatal(err)
	}
	assertRestricted(0) // Revocation is effective on the following request.
}
