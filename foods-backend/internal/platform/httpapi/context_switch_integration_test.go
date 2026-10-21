package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPlatformAdminCanLoadContextAfterSwitchingTenant(t *testing.T) {
	pool := integrationPool(t)
	source := seedInventoryScope(t, pool)
	target := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()

	password := "PlatformAdminPass123"
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatal(err)
	}

	var email string
	if err = pool.QueryRow(ctx, `
		UPDATE users
		SET platform_admin=true,password_hash=$2
		WHERE id=$1
		RETURNING email
	`, source.UserID, string(hash)).Scan(&email); err != nil {
		t.Fatal(err)
	}

	var targetProfileID string
	if err = pool.QueryRow(ctx, `
		SELECT id
		FROM organization_fiscal_profiles
		WHERE organization_id=$1 AND is_default AND active
		LIMIT 1
	`, target.OrganizationID).Scan(&targetProfileID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `
		UPDATE locations
		SET fiscal_profile_id=$1
		WHERE id=$2 AND organization_id=$3
	`, targetProfileID, target.LocationID, target.OrganizationID); err != nil {
		t.Fatal(err)
	}

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/login",
		bytes.NewReader([]byte(
			`{"email":"` + email + `","password":"` + password + `"}`,
		)),
	)
	loginRec := httptest.NewRecorder()
	api.Routes().ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf(
			"platform login: %d %s",
			loginRec.Code,
			loginRec.Body.String(),
		)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("platform login did not create session cookie")
	}

	switchBody, err := json.Marshal(switchContextInput{
		OrganizationID: target.OrganizationID,
		LocationID:     target.LocationID,
	})
	if err != nil {
		t.Fatal(err)
	}
	switchReq := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/switch-context",
		bytes.NewReader(switchBody),
	)
	switchReq.AddCookie(cookies[0])
	switchRec := httptest.NewRecorder()
	api.Routes().ServeHTTP(switchRec, switchReq)
	if switchRec.Code != http.StatusOK {
		t.Fatalf(
			"switch context: %d %s",
			switchRec.Code,
			switchRec.Body.String(),
		)
	}

	contextReq := httptest.NewRequest(
		http.MethodGet,
		"/v1/admin/context",
		nil,
	)
	contextReq.AddCookie(cookies[0])
	contextRec := httptest.NewRecorder()
	api.Routes().ServeHTTP(contextRec, contextReq)
	if contextRec.Code != http.StatusOK {
		t.Fatalf(
			"context after switch: %d %s",
			contextRec.Code,
			contextRec.Body.String(),
		)
	}

	var got struct {
		User struct {
			PlatformAdmin bool `json:"platformAdmin"`
		} `json:"user"`
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
		Location struct {
			ID string `json:"id"`
		} `json:"location"`
		Permissions []string `json:"permissions"`
		MenuAccess  []string `json:"menuAccess"`
	}
	if err = json.Unmarshal(
		contextRec.Body.Bytes(),
		&got,
	); err != nil {
		t.Fatal(err)
	}

	if !got.User.PlatformAdmin {
		t.Fatal("switched context lost platformAdmin flag")
	}
	if got.Organization.ID != target.OrganizationID {
		t.Fatalf(
			"expected target organization %s, got %s",
			target.OrganizationID,
			got.Organization.ID,
		)
	}
	if got.Location.ID != target.LocationID {
		t.Fatalf(
			"expected target location %s, got %s",
			target.LocationID,
			got.Location.ID,
		)
	}
	if len(got.Permissions) != 1 || got.Permissions[0] != "*" {
		t.Fatalf(
			"platform admin permissions changed after switch: %v",
			got.Permissions,
		)
	}
	if len(got.MenuAccess) != 1 || got.MenuAccess[0] != "*" {
		t.Fatalf(
			"platform admin menu access changed after switch: %v",
			got.MenuAccess,
		)
	}

	var sessionOrg, sessionLocation string
	sum := sha256Sum(cookies[0].Value)
	if err = pool.QueryRow(ctx, `
		SELECT organization_id,location_id
		FROM sessions
		WHERE token_hash=$1
		  AND revoked_at IS NULL
		  AND expires_at>$2
	`, sum, time.Now()).Scan(
		&sessionOrg,
		&sessionLocation,
	); err != nil {
		t.Fatal(err)
	}
	if sessionOrg != target.OrganizationID ||
		sessionLocation != target.LocationID {
		t.Fatalf(
			"session context was not persisted: org=%s location=%s",
			sessionOrg,
			sessionLocation,
		)
	}
}
