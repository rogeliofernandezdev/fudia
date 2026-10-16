package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func seedUserCapacity(t *testing.T, max *int) (*API, scope, string) {
	t.Helper()
	api, s, roleID := seedIdentityAdministrator(t)
	ctx := context.Background()
	var planID string
	if err := api.db.QueryRow(ctx, `
		INSERT INTO subscription_plans(code,name,currency,monthly_price,annual_price,max_users,module_keys,terms_version)
		VALUES($1,'Quota Test','PEN',1,10,$2,ARRAY['usuarios','locales'],'test') RETURNING id
	`, fmt.Sprintf("quota-%d", time.Now().UnixNano()), max).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	tx, err := api.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = createOrganizationSubscription(ctx, tx, s.OrganizationID, s.UserID, planID, "monthly", true); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := api.db.Exec(ctx, `DELETE FROM organization_subscriptions WHERE organization_id=$1`, s.OrganizationID); err != nil {
			t.Errorf("clean quota subscription: %v", err)
		}
		if _, err := api.db.Exec(ctx, `DELETE FROM subscription_plans WHERE id=$1`, planID); err != nil {
			t.Errorf("clean quota plan: %v", err)
		}
	})
	return api, s, roleID
}

func seedQuotaUser(t *testing.T, api *API, s scope, active bool) string {
	t.Helper()
	var id string
	if err := api.db.QueryRow(context.Background(), `
		INSERT INTO users(organization_id,email,full_name,password_hash,active)
		VALUES($1,$2,'Quota User','unused', $3) RETURNING id
	`, s.OrganizationID, fmt.Sprintf("quota-%d@example.test", time.Now().UnixNano()), active).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func quotaStatus(api *API, s scope, id string, active bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest("PATCH", "/v1/admin/users/"+id+"/status", bytes.NewBufferString(fmt.Sprintf(`{"active":%t}`, active)))
	req.SetPathValue("id", id)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateUserStatus(rec, req)
	return rec
}

func quotaCreate(api *API, s scope, roleID string) *httptest.ResponseRecorder {
	body := fmt.Sprintf(`{"fullName":"New Quota User","email":"quota-create-%d@example.test","password":"QuotaPass123","assignments":[{"roleId":%q,"locationId":%q}]}`, time.Now().UnixNano(), roleID, s.LocationID)
	req := httptest.NewRequest("POST", "/v1/admin/users", bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.createUser(rec, req)
	return rec
}

func TestUserCapacityCreationReactivationAndEditing(t *testing.T) {
	max := 2
	api, s, roleID := seedUserCapacity(t, &max)
	active := seedQuotaUser(t, api, s, true)
	inactive := seedQuotaUser(t, api, s, false)
	for _, rec := range []*httptest.ResponseRecorder{quotaCreate(api, s, roleID), quotaStatus(api, s, inactive, true)} {
		if rec.Code != 409 || !strings.Contains(rec.Body.String(), "plan_limit_reached") {
			t.Fatalf("expected quota rejection: %d %s", rec.Code, rec.Body.String())
		}
	}
	var remainsInactive bool
	if err := api.db.QueryRow(context.Background(), `SELECT NOT active FROM users WHERE id=$1`, inactive).Scan(&remainsInactive); err != nil {
		t.Fatal(err)
	}
	if !remainsInactive {
		t.Fatal("rejected activation modified the user")
	}
	if rec := quotaStatus(api, s, active, true); rec.Code != 204 {
		t.Fatalf("idempotent active update: %d %s", rec.Code, rec.Body.String())
	}
	var email string
	if err := api.db.QueryRow(context.Background(), `SELECT email FROM users WHERE id=$1`, active).Scan(&email); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"fullName":"Edited at capacity","email":%q,"assignments":[{"roleId":%q,"locationId":%q}]}`, email, roleID, s.LocationID)
	req := httptest.NewRequest("PATCH", "/v1/admin/users/"+active, bytes.NewBufferString(body))
	req.SetPathValue("id", active)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.updateUser(rec, req)
	if rec.Code != 200 {
		t.Fatalf("edit must remain allowed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := quotaStatus(api, s, active, false); rec.Code != 204 {
		t.Fatalf("deactivate: %d %s", rec.Code, rec.Body.String())
	}
	if rec := quotaStatus(api, s, inactive, true); rec.Code != 204 {
		t.Fatalf("activate after freeing capacity: %d %s", rec.Code, rec.Body.String())
	}
	otherAPI, other, _ := seedUserCapacity(t, &max)
	if rec := quotaStatus(otherAPI, other, inactive, true); rec.Code != 404 {
		t.Fatalf("cross-tenant activation: %d %s", rec.Code, rec.Body.String())
	}
}

func TestUserCapacityConcurrentActivationAndCreation(t *testing.T) {
	for _, withCreation := range []bool{false, true} {
		t.Run(fmt.Sprintf("creation=%t", withCreation), func(t *testing.T) {
			max := 2
			api, s, roleID := seedUserCapacity(t, &max)
			first := seedQuotaUser(t, api, s, false)
			second := seedQuotaUser(t, api, s, false)
			start := make(chan struct{})
			results := make(chan *httptest.ResponseRecorder, 2)
			go func() { <-start; results <- quotaStatus(api, s, first, true) }()
			go func() {
				<-start
				if withCreation {
					results <- quotaCreate(api, s, roleID)
				} else {
					results <- quotaStatus(api, s, second, true)
				}
			}()
			close(start)
			allowed, rejected := 0, 0
			for i := 0; i < 2; i++ {
				rec := <-results
				if rec.Code == 204 || rec.Code == 201 {
					allowed++
				} else if rec.Code == 409 && strings.Contains(rec.Body.String(), "plan_limit_reached") {
					rejected++
				} else {
					t.Fatalf("unexpected concurrent response: %d %s", rec.Code, rec.Body.String())
				}
			}
			if allowed != 1 || rejected != 1 {
				t.Fatalf("allowed=%d rejected=%d", allowed, rejected)
			}
			var count int
			if err := api.db.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, s.OrganizationID).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != max {
				t.Fatalf("active users=%d max=%d", count, max)
			}
		})
	}
}

func TestUserCapacityUnlimitedAndUnavailableSubscription(t *testing.T) {
	api, s, roleID := seedUserCapacity(t, nil)
	inactive := seedQuotaUser(t, api, s, false)
	if rec := quotaStatus(api, s, inactive, true); rec.Code != 204 {
		t.Fatalf("unlimited activation: %d %s", rec.Code, rec.Body.String())
	}
	if rec := quotaCreate(api, s, roleID); rec.Code != 201 {
		t.Fatalf("unlimited creation: %d %s", rec.Code, rec.Body.String())
	}
	missing, other, _ := seedIdentityAdministrator(t)
	target := seedQuotaUser(t, missing, other, false)
	if rec := quotaStatus(missing, other, target, true); rec.Code != 503 || !strings.Contains(rec.Body.String(), "subscription_unavailable") {
		t.Fatalf("missing subscription: %d %s", rec.Code, rec.Body.String())
	}
	var active bool
	if err := missing.db.QueryRow(context.Background(), `SELECT active FROM users WHERE id=$1`, target).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active {
		t.Fatal("activation without subscription must not commit")
	}
}

func TestUserCapacityUsageExcludesPlatformAccounts(t *testing.T) {
	max := 2
	api, s, roleID := seedUserCapacity(t, &max)
	platform := seedQuotaUser(t, api, s, true)
	if _, err := api.db.Exec(context.Background(), `UPDATE users SET platform_admin=true WHERE id=$1`, platform); err != nil {
		t.Fatal(err)
	}
	target := seedQuotaUser(t, api, s, false)
	if rec := quotaStatus(api, s, target, true); rec.Code != 204 {
		t.Fatalf("platform account must not consume capacity: %d %s", rec.Code, rec.Body.String())
	}
	if rec := quotaCreate(api, s, roleID); rec.Code != 409 {
		t.Fatalf("tenant administrator and active user consume two slots: %d %s", rec.Code, rec.Body.String())
	}
	if rec := quotaStatus(api, s, platform, false); rec.Code != 403 {
		t.Fatalf("platform protection must remain: %d %s", rec.Code, rec.Body.String())
	}
	req := httptest.NewRequest("GET", "/v1/admin/subscription", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, s))
	rec := httptest.NewRecorder()
	api.getOrganizationSubscription(rec, req)
	var result struct{ Usage struct{ Users int } }
	if rec.Code != 200 {
		t.Fatalf("subscription usage: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Usage.Users != max {
		t.Fatalf("usage users=%d expected=%d", result.Usage.Users, max)
	}
}
