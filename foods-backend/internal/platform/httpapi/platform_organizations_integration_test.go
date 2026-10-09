package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestPlatformOrganizationsReportPlanAndPaymentStanding(t *testing.T) {
	pool := integrationPool(t)
	actor := seedInventoryScope(t, pool)
	api := New(pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()

	var planID, tradeName string
	if err := pool.QueryRow(ctx, `
		INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,module_keys,terms_version,active)
		VALUES($1,'Plan listado','Plan de prueba','PEN',120,1200,0,ARRAY['dashboard'],'test-v1',true)
		RETURNING id
	`, fmt.Sprintf("list-%d", nonce)).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT trade_name FROM organizations WHERE id=$1`, actor.OrganizationID).Scan(&tradeName); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_subscriptions(organization_id,plan_id,billing_cycle,price_amount,currency,status,current_period_starts_at,current_period_ends_at,renews_at)
		VALUES($1,$2,'monthly',120,'PEN','active',now()-interval '27 days',now()+interval '3 days',now()+interval '3 days')
		ON CONFLICT(organization_id) DO UPDATE SET plan_id=excluded.plan_id,status='active',price_amount=120,currency='PEN',renews_at=excluded.renews_at,current_period_ends_at=excluded.current_period_ends_at
	`, actor.OrganizationID, planID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM organization_subscriptions WHERE organization_id=$1`, actor.OrganizationID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM subscription_plans WHERE id=$1`, planID)
	})

	type listResponse struct {
		Items   []platformOrganizationView `json:"items"`
		Total   int                        `json:"total"`
		Summary map[string]int             `json:"summary"`
	}
	list := func(query url.Values) listResponse {
		t.Helper()
		req := httptest.NewRequest("GET", "/v1/platform/organizations?"+query.Encode(), nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
		rec := httptest.NewRecorder()
		api.listPlatformOrganizations(rec, req)
		if rec.Code != 200 {
			t.Fatalf("list organizations: %d %s", rec.Code, rec.Body.String())
		}
		var out listResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	found := list(url.Values{"q": {tradeName}, "planId": {planID}})
	if found.Total != 1 || len(found.Items) != 1 {
		t.Fatalf("expected one organization, got %d", found.Total)
	}
	item := found.Items[0]
	if item.Subscription == nil || item.Subscription.Plan.Name != "Plan listado" || item.Subscription.PriceAmount != "120.00" {
		t.Fatalf("unexpected subscription: %#v", item.Subscription)
	}
	if item.PaymentStanding != "due_soon" {
		t.Fatalf("renewal within the window must be due_soon, got %s", item.PaymentStanding)
	}
	if found.Summary["total"] < 1 || found.Summary["due_soon"] < 1 {
		t.Fatalf("summary must count the organization: %#v", found.Summary)
	}

	if _, err := pool.Exec(ctx, `UPDATE organization_subscriptions SET renews_at=now()-interval '1 day' WHERE organization_id=$1`, actor.OrganizationID); err != nil {
		t.Fatal(err)
	}
	overdue := list(url.Values{"q": {tradeName}, "standing": {"overdue"}})
	if overdue.Total != 1 || overdue.Items[0].PaymentStanding != "overdue" || overdue.Items[0].Subscription.Status != "past_due" {
		t.Fatalf("expired renewal must be reported overdue: %#v", overdue.Items)
	}

	req := httptest.NewRequest("GET", "/v1/platform/organizations?standing=unknown", nil)
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, actor))
	rec := httptest.NewRecorder()
	api.listPlatformOrganizations(rec, req)
	if rec.Code != 400 {
		t.Fatalf("invalid standing must be rejected, got %d", rec.Code)
	}

	// Plataforma gestiona la suscripción por id sin depender de la empresa de la sesión.
	platformActor := seedInventoryScope(t, pool)
	byID := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.SetPathValue("id", actor.OrganizationID)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, platformActor))
		rec := httptest.NewRecorder()
		switch method {
		case "GET":
			api.getOrganizationSubscription(rec, req)
		case "POST":
			api.recordSubscriptionPayment(rec, req)
		}
		return rec
	}
	read := byID("GET", "/v1/platform/organizations/"+actor.OrganizationID+"/subscription", "")
	if read.Code != 200 {
		t.Fatalf("read subscription by id: %d %s", read.Code, read.Body.String())
	}
	var detail organizationSubscriptionView
	if err := json.Unmarshal(read.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Organization.ID != actor.OrganizationID || detail.Organization.TradeName != tradeName || detail.Plan.ID != planID {
		t.Fatalf("subscription must belong to the requested organization: %#v", detail.Organization)
	}
	paid := byID("POST", "/v1/platform/organizations/"+actor.OrganizationID+"/subscription/payments", `{"amount":"120.00","currency":"PEN","status":"paid","provider":"manual","externalReference":"`+fmt.Sprintf("list-%d", nonce)+`","paidAt":""}`)
	if paid.Code != 201 {
		t.Fatalf("record payment by id: %d %s", paid.Code, paid.Body.String())
	}
	var platformPayments int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM subscription_payments WHERE organization_id=$1`, platformActor.OrganizationID).Scan(&platformPayments); err != nil {
		t.Fatal(err)
	}
	if platformPayments != 0 {
		t.Fatalf("payment must not be recorded on the session organization")
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM subscription_payments WHERE organization_id=$1`, actor.OrganizationID)
	})
	upToDate := list(url.Values{"q": {tradeName}})
	if upToDate.Items[0].PaymentStanding != "up_to_date" || upToDate.Items[0].LastPayment == nil || upToDate.Items[0].LastPayment.Amount != "120.00" {
		t.Fatalf("a confirmed payment must leave the organization up to date: %#v", upToDate.Items[0])
	}

	detailReq := httptest.NewRequest("GET", "/v1/platform/organizations/"+actor.OrganizationID, nil)
	detailReq.SetPathValue("id", actor.OrganizationID)
	detailReq = detailReq.WithContext(context.WithValue(detailReq.Context(), scopeKey{}, platformActor))
	detailRec := httptest.NewRecorder()
	api.getPlatformOrganization(detailRec, detailReq)
	if detailRec.Code != 200 {
		t.Fatalf("organization detail: %d %s", detailRec.Code, detailRec.Body.String())
	}
	var company platformOrganizationDetail
	if err := json.Unmarshal(detailRec.Body.Bytes(), &company); err != nil {
		t.Fatal(err)
	}
	if company.TradeName != tradeName || company.Subscription == nil || company.Subscription.PlanName != "Plan listado" || company.Subscription.ModuleCount != 1 {
		t.Fatalf("unexpected organization detail: %#v", company.Subscription)
	}
	if company.Usage.Locations != 1 || len(company.Locations) != 1 || company.Payments.PaidCount != 1 || company.Payments.TotalPaid != "120.00" || company.PaymentStanding != "up_to_date" {
		t.Fatalf("detail must report usage, locations and payments: %#v %#v", company.Usage, company.Payments)
	}

	missing := httptest.NewRequest("GET", "/v1/platform/organizations/not-a-uuid/subscription", nil)
	missing.SetPathValue("id", "not-a-uuid")
	missing = missing.WithContext(context.WithValue(missing.Context(), scopeKey{}, platformActor))
	missingRec := httptest.NewRecorder()
	api.getOrganizationSubscription(missingRec, missing)
	if missingRec.Code != 404 {
		t.Fatalf("unknown organization must return 404, got %d", missingRec.Code)
	}
}
