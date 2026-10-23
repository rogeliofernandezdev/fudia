package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type conciergeUsageFixture struct {
	pool   *pgxpool.Pool
	api    *integrationAPI
	scope  scope
	token  string
	planID string
}

// seedConciergeUsage creates a company with Concierge active, one QR table and
// a plan capped at `cap` conversations. The global policy is restored after.
func seedConciergeUsage(t *testing.T, cap int) conciergeUsageFixture {
	t.Helper()
	pool := integrationPool(t)
	s := seedInventoryScope(t, pool)
	ctx := context.Background()
	nonce := time.Now().UnixNano()
	t.Setenv("FUDIA_CONCIERGE_API_KEY", "concierge-test-key")
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,'whatsapp_bot',true)
		ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now()
	`, s.OrganizationID); err != nil {
		t.Fatal(err)
	}
	var token, planID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		VALUES($1,$2,$3,4,'Principal',true,encode(gen_random_bytes(16),'hex'),true)
		RETURNING qr_token
	`, s.OrganizationID, s.LocationID, fmt.Sprintf("Mesa uso %d", nonce)).Scan(&token); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,module_keys,terms_version,active,max_concierge_conversations)
		VALUES($1,'Plan Concierge','Plan de prueba','PEN',89,890,0,ARRAY['whatsapp_bot'],'test-v1',true,$2)
		RETURNING id
	`, fmt.Sprintf("concierge-%d", nonce), cap).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO organization_subscriptions(organization_id,plan_id,billing_cycle,price_amount,currency,status,current_period_starts_at,current_period_ends_at,renews_at)
		VALUES($1,$2,'monthly',89,'PEN','active',now(),now()+interval '30 days',now()+interval '30 days')
	`, s.OrganizationID, planID); err != nil {
		t.Fatal(err)
	}
	var mode string
	var warn int
	if err := pool.QueryRow(ctx, `SELECT mode,warn_percent FROM concierge_usage_settings`).Scan(&mode, &warn); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `UPDATE concierge_usage_settings SET mode=$1,warn_percent=$2`, mode, warn)
		_, _ = pool.Exec(bg, `DELETE FROM concierge_token_usage WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM concierge_conversations WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM concierge_monthly_usage WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM organization_concierge_limits WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM audit_log WHERE action='concierge.limit.updated' AND entity_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM organization_subscriptions WHERE organization_id=$1`, s.OrganizationID)
		_, _ = pool.Exec(bg, `DELETE FROM subscription_plans WHERE id=$1`, planID)
	})
	return conciergeUsageFixture{pool: pool, api: New(pool), scope: s, token: token, planID: planID}
}

func (f conciergeUsageFixture) integration(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+f.token+path, bytes.NewBufferString(body))
	req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
	rec := httptest.NewRecorder()
	f.api.Routes().ServeHTTP(rec, req)
	return rec
}

func (f conciergeUsageFixture) open(t *testing.T, conversationID, kind string) *httptest.ResponseRecorder {
	t.Helper()
	return f.integration(t, "/conversations", fmt.Sprintf(`{"conversationId":%q,"kind":%q}`, conversationID, kind))
}

// platform calls the handler as the platform administrator would after the
// route's platform-admin check (covered by TestConciergeUsageIsPlatformOnly).
func (f conciergeUsageFixture) platform(t *testing.T, handler func(http.ResponseWriter, *http.Request), method, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/v1/platform/concierge-usage", bytes.NewBufferString(body))
	req.SetPathValue("id", f.scope.OrganizationID)
	q := req.URL.Query()
	q.Add("organizationId", f.scope.OrganizationID)
	req.URL.RawQuery = q.Encode()
	req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, f.scope))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

type usageSummaryView struct {
	OrganizationID        string  `json:"organizationId"`
	Month                 string  `json:"month"`
	MaxConversations      *int    `json:"maxConversations"`
	LimitSource           string  `json:"limitSource"`
	Conversations         int     `json:"conversations"`
	RejectedConversations int     `json:"rejectedConversations"`
	UsagePercent          *int    `json:"usagePercent"`
	Alert                 string  `json:"alert"`
	PlanName              *string `json:"planName"`
}

func (f conciergeUsageFixture) summary(t *testing.T) (usageSummaryView, map[string]int) {
	t.Helper()
	rec := f.platform(t, f.api.listConciergeUsage, "GET", "")
	if rec.Code != 200 {
		t.Fatalf("list usage: %d %s", rec.Code, rec.Body.String())
	}
	var page struct {
		Summary map[string]int     `json:"summary"`
		Items   []usageSummaryView `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].OrganizationID != f.scope.OrganizationID {
		t.Fatalf("filter by organizationId must return only that company: %+v", page.Items)
	}
	return page.Items[0], page.Summary
}

func (f conciergeUsageFixture) settings(t *testing.T, mode string, warn int) {
	t.Helper()
	rec := f.platform(t, f.api.updateConciergeUsageSettings, "PATCH", fmt.Sprintf(`{"mode":%q,"warnPercent":%d}`, mode, warn))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"mode":"`+mode+`"`) {
		t.Fatalf("save settings: %d %s", rec.Code, rec.Body.String())
	}
}

func (f conciergeUsageFixture) override(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	return f.platform(t, f.api.updateOrganizationConciergeLimit, "PATCH", body)
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, what string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("%s: expected %d, got %d %s", what, status, rec.Code, rec.Body.String())
	}
}

func TestConciergeUsageMeasuresWarnsAndEnforcesWithoutRedeploy(t *testing.T) {
	f := seedConciergeUsage(t, 2)
	f.settings(t, "measure", 80)

	// A conversation counts once; a retry with the same conversationId does not.
	expectStatus(t, f.open(t, "conv-a", "table"), 201, "open A")
	expectStatus(t, f.open(t, "conv-a", "table"), 200, "reopen A")
	expectStatus(t, f.open(t, "conv-b", "delivery"), 201, "open B")
	// Measuring never blocks, even past the cap, and raises no alert.
	expectStatus(t, f.open(t, "conv-c", "table"), 201, "open C past cap while measuring")
	got, _ := f.summary(t)
	if got.Conversations != 3 || got.MaxConversations == nil || *got.MaxConversations != 2 || got.LimitSource != "plan" ||
		got.UsagePercent == nil || *got.UsagePercent != 150 || got.Alert != "none" || got.PlanName == nil || *got.PlanName != "Plan Concierge" {
		t.Fatalf("unexpected measured usage: %+v", got)
	}
	if _, err := time.Parse("2006-01", got.Month); err != nil {
		t.Fatalf("month must be YYYY-MM: %q", got.Month)
	}

	// Warn mode: at or past the warn percentage the platform administrator sees it.
	f.settings(t, "warn", 50)
	expectStatus(t, f.override(t, `{"limitSource":"custom","maxConversations":5}`), 200, "custom cap")
	got, summary := f.summary(t)
	if got.Alert != "warning" || *got.UsagePercent != 60 || got.LimitSource != "custom" || *got.MaxConversations != 5 || summary["warning"] < 1 {
		t.Fatalf("expected warning at 60%%: %+v %v", got, summary)
	}
	expectStatus(t, f.open(t, "conv-d", "table"), 201, "warn mode never blocks")
	expectStatus(t, f.open(t, "conv-e", "table"), 201, "warn mode never blocks at 100%")
	if got, summary = f.summary(t); got.Alert != "reached" || summary["reached"] < 1 {
		t.Fatalf("expected reached alert: %+v %v", got, summary)
	}

	// Enforce: new conversations are rejected; the ones in progress continue.
	f.settings(t, "enforce", 80)
	rejected := f.open(t, "conv-f", "table")
	expectStatus(t, rejected, 409, "enforce at cap")
	if !strings.Contains(rejected.Body.String(), "concierge_conversation_limit_reached") {
		t.Fatalf("rejection must be explicit: %s", rejected.Body.String())
	}
	expectStatus(t, f.open(t, "conv-a", "table"), 200, "conversation in progress is never cut")
	expectStatus(t, f.integration(t, "/conversations/conv-a/usage", `{"requestId":"turn-1","tiers":[{"tier":"lite","inputTokens":100,"outputTokens":20},{"tier":"flash","inputTokens":50,"outputTokens":10}]}`), 204, "usage of open conversation")
	expectStatus(t, f.integration(t, "/conversations/conv-a/usage", `{"requestId":"turn-1","tiers":[{"tier":"lite","inputTokens":100,"outputTokens":20}]}`), 204, "repeated report")
	expectStatus(t, f.integration(t, "/conversations/conv-a/usage", `{"requestId":"turn-2","tiers":[{"tier":"lite","inputTokens":7,"outputTokens":3}]}`), 204, "second turn")
	if got, _ = f.summary(t); got.Conversations != 5 || got.RejectedConversations != 1 {
		t.Fatalf("rejection must not count as conversation: %+v", got)
	}

	// Changing the exception or the plan cap applies immediately.
	expectStatus(t, f.override(t, `{"limitSource":"unlimited"}`), 200, "unlimited")
	expectStatus(t, f.open(t, "conv-f", "table"), 201, "unlimited exception opens")
	if got, _ = f.summary(t); got.MaxConversations != nil || got.UsagePercent != nil || got.Alert != "none" || got.LimitSource != "unlimited" {
		t.Fatalf("unlimited exception: %+v", got)
	}
	expectStatus(t, f.override(t, `{"limitSource":"plan"}`), 200, "back to plan")
	expectStatus(t, f.open(t, "conv-g", "table"), 409, "plan cap of 2 blocks again")

	var planCode string
	if err := f.pool.QueryRow(context.Background(), `SELECT code FROM subscription_plans WHERE id=$1`, f.planID).Scan(&planCode); err != nil {
		t.Fatal(err)
	}
	planReq := httptest.NewRequest("PATCH", "/v1/platform/plans/"+f.planID, bytes.NewBufferString(fmt.Sprintf(
		`{"code":%q,"name":"Plan Concierge","description":"Plan de prueba","currency":"PEN","monthlyPrice":"89","annualPrice":"890","trialDays":0,"maxLocations":null,"maxUsers":null,"maxConciergeConversations":10,"moduleKeys":["whatsapp_bot"],"termsVersion":"test-v1","active":true}`,
		planCode)))
	planReq.SetPathValue("id", f.planID)
	planReq = planReq.WithContext(context.WithValue(planReq.Context(), scopeKey{}, f.scope))
	planRec := httptest.NewRecorder()
	f.api.updateSubscriptionPlan(planRec, planReq)
	if planRec.Code != 200 || !strings.Contains(planRec.Body.String(), `"maxConciergeConversations":10`) {
		t.Fatalf("update plan cap: %d %s", planRec.Code, planRec.Body.String())
	}
	expectStatus(t, f.open(t, "conv-g", "table"), 201, "raised plan cap opens")

	// A report for a conversation opened elsewhere never blocks and counts it.
	expectStatus(t, f.override(t, `{"limitSource":"custom","maxConversations":1}`), 200, "tight cap")
	expectStatus(t, f.integration(t, "/conversations/conv-legacy/usage", `{"requestId":"turn-1","tiers":[{"tier":"flash","inputTokens":1,"outputTokens":1}]}`), 204, "unregistered conversation report")

	detailRec := f.platform(t, f.api.getOrganizationConciergeUsage, "GET", "")
	expectStatus(t, detailRec, 200, "detail")
	var detail struct {
		usageSummaryView
		PlanMaxConversations *int `json:"planMaxConversations"`
		Tokens               []struct {
			Tier         string `json:"tier"`
			InputTokens  int64  `json:"inputTokens"`
			OutputTokens int64  `json:"outputTokens"`
		} `json:"tokens"`
		History []struct {
			Month         string `json:"month"`
			Conversations int    `json:"conversations"`
			InputTokens   int64  `json:"inputTokens"`
			OutputTokens  int64  `json:"outputTokens"`
		} `json:"history"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Conversations != 8 || detail.PlanMaxConversations == nil || *detail.PlanMaxConversations != 10 || detail.LimitSource != "custom" || detail.Alert != "reached" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if len(detail.Tokens) != 2 || detail.Tokens[0].Tier != "lite" || detail.Tokens[0].InputTokens != 107 || detail.Tokens[0].OutputTokens != 23 ||
		detail.Tokens[1].Tier != "flash" || detail.Tokens[1].InputTokens != 51 || detail.Tokens[1].OutputTokens != 11 {
		t.Fatalf("tokens must be idempotent per turn and tier: %+v", detail.Tokens)
	}
	if len(detail.History) != 1 || detail.History[0].Month != detail.Month || detail.History[0].Conversations != 8 || detail.History[0].InputTokens != 158 {
		t.Fatalf("unexpected history: %+v", detail.History)
	}

	// Invalid input is rejected explicitly.
	expectStatus(t, f.open(t, "conv-x", "chat"), 400, "invalid kind")
	expectStatus(t, f.integration(t, "/conversations/conv-a/usage", `{"requestId":"turn-9","tiers":[{"tier":"pro","inputTokens":1,"outputTokens":1}]}`), 400, "invalid tier")
	expectStatus(t, f.platform(t, f.api.updateConciergeUsageSettings, "PATCH", `{"mode":"block","warnPercent":80}`), 400, "invalid mode")
	expectStatus(t, f.platform(t, f.api.updateConciergeUsageSettings, "PATCH", `{"mode":"warn","warnPercent":0}`), 400, "invalid percent")
	expectStatus(t, f.override(t, `{"limitSource":"custom"}`), 400, "custom without cap")
	missing := httptest.NewRequest("GET", "/", nil)
	missing.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	missingRec := httptest.NewRecorder()
	f.api.getOrganizationConciergeUsage(missingRec, missing.WithContext(context.WithValue(missing.Context(), scopeKey{}, f.scope)))
	expectStatus(t, missingRec, 404, "unknown company")
}

func TestConciergeUsageEnforcementAllowsOnlyTheCapUnderConcurrency(t *testing.T) {
	f := seedConciergeUsage(t, 3)
	f.settings(t, "enforce", 80)
	results := make(chan int, 8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			req := httptest.NewRequest("POST", "/v1/integrations/concierge/"+f.token+"/conversations", bytes.NewBufferString(fmt.Sprintf(`{"conversationId":"race-%d","kind":"table"}`, i)))
			req.Header.Set("X-Fudia-Concierge-Key", "concierge-test-key")
			rec := httptest.NewRecorder()
			f.api.Routes().ServeHTTP(rec, req)
			results <- rec.Code
		}(i)
	}
	created, rejected := 0, 0
	for i := 0; i < 8; i++ {
		switch <-results {
		case 201:
			created++
		case 409:
			rejected++
		}
	}
	if created != 3 || rejected != 5 {
		t.Fatalf("expected exactly 3 opened and 5 rejected, got %d/%d", created, rejected)
	}
	if got, _ := f.summary(t); got.Conversations != 3 || got.RejectedConversations != 5 {
		t.Fatalf("counter drifted: %+v", got)
	}
}

func TestConciergeUsageIsPlatformOnly(t *testing.T) {
	f := seedConciergeUsage(t, 500)
	ctx := context.Background()
	// A real company session: authenticated, not platform administrator.
	sessionToken := fmt.Sprintf("company-session-%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(sessionToken))
	if _, err := f.pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,organization_id,location_id,expires_at) VALUES($1,$2,$3,$4,now()+interval '1 hour')`,
		hash[:], f.scope.UserID, f.scope.OrganizationID, f.scope.LocationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM sessions WHERE token_hash=$1`, hash[:]) })
	org := f.scope.OrganizationID
	for _, call := range []struct{ method, path, body string }{
		{"GET", "/v1/platform/concierge-usage", ""},
		{"GET", "/v1/platform/concierge-usage?organizationId=" + org, ""},
		{"GET", "/v1/platform/concierge-usage/settings", ""},
		{"PATCH", "/v1/platform/concierge-usage/settings", `{"mode":"enforce","warnPercent":10}`},
		{"GET", "/v1/platform/organizations/" + org + "/concierge-usage", ""},
		{"PATCH", "/v1/platform/organizations/" + org + "/concierge-usage", `{"limitSource":"unlimited"}`},
		{"GET", "/v1/platform/plans", ""},
		{"PATCH", "/v1/platform/plans/" + f.planID, `{"maxConciergeConversations":1}`},
	} {
		req := httptest.NewRequest(call.method, call.path, bytes.NewBufferString(call.body))
		req.AddCookie(&http.Cookie{Name: "foods_session", Value: sessionToken})
		rec := httptest.NewRecorder()
		f.api.Routes().ServeHTTP(rec, req)
		if rec.Code != 403 || !strings.Contains(rec.Body.String(), "platform_forbidden") {
			t.Fatalf("%s %s must be forbidden for a company administrator: %d %s", call.method, call.path, rec.Code, rec.Body.String())
		}
	}
	var mode string
	var exceptions int
	if err := f.pool.QueryRow(ctx, `SELECT mode,(SELECT count(*) FROM organization_concierge_limits WHERE organization_id=$1) FROM concierge_usage_settings`, org).Scan(&mode, &exceptions); err != nil {
		t.Fatal(err)
	}
	if mode == "enforce" || exceptions != 0 {
		t.Fatalf("forbidden writes must not apply: mode=%s exceptions=%d", mode, exceptions)
	}

	expectStatus(t, f.open(t, "conv-visible", "table"), 201, "open conversation")
	// Company-facing responses never carry caps, mode or usage.
	for name, handler := range map[string]func(http.ResponseWriter, *http.Request){
		"context":            f.api.getContext,
		"subscription":       f.api.getOrganizationSubscription,
		"available plans":    f.api.listAvailableSubscriptionPlans,
		"concierge settings": f.api.getConciergeSettings,
	} {
		req := httptest.NewRequest("GET", "/v1/admin/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), scopeKey{}, f.scope))
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", name, rec.Code, rec.Body.String())
		}
		for _, leak := range []string{"maxConciergeConversations", "conciergeUsage", "warnPercent", "limitSource", "rejectedConversations", "max_concierge"} {
			if strings.Contains(rec.Body.String(), leak) {
				t.Fatalf("%s must not expose %s: %s", name, leak, rec.Body.String())
			}
		}
	}

	// Platform catalog carries the migrated base caps.
	plansRec := f.platform(t, f.api.listSubscriptionPlans, "GET", "")
	expectStatus(t, plansRec, 200, "platform plans")
	var plans struct {
		Items []struct {
			Code                      string `json:"code"`
			MaxConciergeConversations *int   `json:"maxConciergeConversations"`
		} `json:"items"`
	}
	if err := json.Unmarshal(plansRec.Body.Bytes(), &plans); err != nil {
		t.Fatal(err)
	}
	caps := map[string]int{"emprende": 500, "impulso": 1500, "escala": 4000}
	for _, plan := range plans.Items {
		if want, ok := caps[plan.Code]; ok && (plan.MaxConciergeConversations == nil || *plan.MaxConciergeConversations != want) {
			t.Fatalf("plan %s must start with cap %d: %v", plan.Code, want, plan.MaxConciergeConversations)
		}
	}
}

func TestConciergeConversationLimitsMigration(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	migration := func(direction string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000091_concierge_conversation_limits."+direction+".sql"))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(migration("down"))
	var remaining int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_name IN ('concierge_usage_settings','organization_concierge_limits','concierge_monthly_usage','concierge_conversations','concierge_token_usage')
	`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("down must drop the usage tables, %d remain", remaining)
	}
	// The grandfathered plan exists only where 000090 found supplied subscribers.
	exec(`
		INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,max_locations,max_users,module_keys,terms_version,active)
		SELECT 'emprende-abastecimiento','Emprende con abastecimiento','',currency,monthly_price,annual_price,trial_days,max_locations,max_users,module_keys,terms_version,false
		FROM subscription_plans WHERE code='emprende'
		ON CONFLICT(code) DO NOTHING
	`)
	exec(migration("up"))
	rows, err := tx.Query(ctx, `SELECT code,max_concierge_conversations FROM subscription_plans WHERE code IN ('emprende','impulso','escala','emprende-abastecimiento','legacy')`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*int{}
	for rows.Next() {
		var code string
		var cap *int
		if err := rows.Scan(&code, &cap); err != nil {
			t.Fatal(err)
		}
		got[code] = cap
	}
	rows.Close()
	for code, want := range map[string]int{"emprende": 500, "impulso": 1500, "escala": 4000, "emprende-abastecimiento": 500} {
		if got[code] == nil || *got[code] != want {
			t.Fatalf("plan %s cap: want %d got %v", code, want, got[code])
		}
	}
	if cap, ok := got["legacy"]; ok && cap != nil {
		t.Fatalf("legacy plan must stay unlimited, got %d", *cap)
	}
	var mode string
	var warn int
	if err := tx.QueryRow(ctx, `SELECT mode,warn_percent FROM concierge_usage_settings`).Scan(&mode, &warn); err != nil {
		t.Fatal(err)
	}
	if mode != "measure" || warn != 80 {
		t.Fatalf("policy must start measuring at 80%%: %s %d", mode, warn)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO concierge_usage_settings(singleton) VALUES(true)`); err == nil {
		t.Fatal("policy must be a single row")
	}
}
