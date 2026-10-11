package httpapi

import (
	"net/http"
	"strings"
)

// Situación de pago derivada de la suscripción; la calcula solo el backend.
const subscriptionDueSoonDays = 7

type platformOrganizationPlan struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type platformOrganizationSubscription struct {
	Plan         platformOrganizationPlan `json:"plan"`
	BillingCycle string                   `json:"billingCycle"`
	PriceAmount  string                   `json:"priceAmount"`
	Currency     string                   `json:"currency"`
	Status       string                   `json:"status"`
	RenewsAt     *string                  `json:"renewsAt"`
	TrialEndsAt  *string                  `json:"trialEndsAt"`
	AutoRenew    bool                     `json:"autoRenew"`
}

type platformOrganizationPayment struct {
	Amount   string  `json:"amount"`
	Currency string  `json:"currency"`
	PaidAt   *string `json:"paidAt"`
}

type platformOrganizationView struct {
	ID                string                            `json:"id"`
	LegalName         string                            `json:"legalName"`
	TradeName         string                            `json:"tradeName"`
	TaxID             string                            `json:"taxId"`
	Active            bool                              `json:"active"`
	CreatedAt         string                            `json:"createdAt"`
	Subscription      *platformOrganizationSubscription `json:"subscription"`
	LastPayment       *platformOrganizationPayment      `json:"lastPayment"`
	PaymentStanding   string                            `json:"paymentStanding"`
}

type optionView struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

var paymentStandingOptions = []optionView{
	{Value: "up_to_date", Label: "Al día"},
	{Value: "due_soon", Label: "Por vencer"},
	{Value: "overdue", Label: "Vencida"},
	{Value: "trial", Label: "En prueba"},
	{Value: "cancelled", Label: "Cancelada"},
	{Value: "none", Label: "Sin suscripción"},
}

func validPaymentStanding(value string) bool {
	for _, option := range paymentStandingOptions {
		if option.Value == value {
			return true
		}
	}
	return false
}

const platformOrganizationsBase = `
	WITH base AS (
		SELECT o.id,o.legal_name,o.trade_name,COALESCE(o.tax_id,'') AS tax_id,o.active,o.created_at,
		       s.id AS subscription_id,p.id AS plan_id,p.code AS plan_code,p.name AS plan_name,
		       s.billing_cycle,s.price_amount,s.currency,s.status,s.renews_at,s.trial_ends_at,s.auto_renew,
		       lp.amount AS last_amount,lp.currency AS last_currency,lp.paid_at AS last_paid_at,
		       CASE
		         WHEN s.id IS NULL THEN 'none'
		         WHEN s.status='cancelled' THEN 'cancelled'
		         WHEN s.status='past_due' THEN 'overdue'
		         WHEN s.status='trial' THEN 'trial'
		         WHEN s.renews_at IS NOT NULL AND s.renews_at < now()+make_interval(days=>$1) THEN 'due_soon'
		         ELSE 'up_to_date'
		       END AS standing
		FROM organizations o
		LEFT JOIN organization_subscriptions s ON s.organization_id=o.id
		LEFT JOIN subscription_plans p ON p.id=s.plan_id
		LEFT JOIN LATERAL (
			SELECT sp.amount,sp.currency,sp.paid_at
			FROM subscription_payments sp
			WHERE sp.organization_id=o.id AND sp.status='paid'
			ORDER BY sp.paid_at DESC NULLS LAST,sp.created_at DESC
			LIMIT 1
		) lp ON true
	)`

// listPlatformOrganizations lista todas las empresas con su plan y situación de pago (solo platform admin).
func (a *API) listPlatformOrganizations(w http.ResponseWriter, r *http.Request) {
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	planID := strings.TrimSpace(r.URL.Query().Get("planId"))
	standing := strings.TrimSpace(r.URL.Query().Get("standing"))
	if standing != "" && !validPaymentStanding(standing) {
		fail(w, 400, "invalid_standing", "La situación de pago no es válida.")
		return
	}

	// Mismo criterio que la lectura de la suscripción: lo vencido pasa a pago pendiente.
	_, _ = a.db.Exec(r.Context(), `
		UPDATE organization_subscriptions
		SET status='past_due',updated_at=now()
		WHERE status IN ('trial','active') AND renews_at IS NOT NULL AND renews_at<now()
	`)

	summary := map[string]int{"total": 0}
	for _, option := range paymentStandingOptions {
		summary[option.Value] = 0
	}
	summaryRows, err := a.db.Query(r.Context(), platformOrganizationsBase+` SELECT standing,count(*) FROM base GROUP BY standing`, subscriptionDueSoonDays)
	if err != nil {
		fail(w, 503, "organizations_unavailable", "No pudimos cargar las empresas.")
		return
	}
	for summaryRows.Next() {
		var key string
		var count int
		if err = summaryRows.Scan(&key, &count); err != nil {
			summaryRows.Close()
			fail(w, 503, "organizations_unavailable", "No pudimos cargar las empresas.")
			return
		}
		summary[key] = count
		summary["total"] += count
	}
	summaryRows.Close()

	rows, err := a.db.Query(r.Context(), platformOrganizationsBase+`
		SELECT id,legal_name,trade_name,tax_id,active,
		       to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       subscription_id IS NOT NULL,COALESCE(plan_id::text,''),COALESCE(plan_code,''),COALESCE(plan_name,''),
		       COALESCE(billing_cycle,''),COALESCE(price_amount::text,''),COALESCE(currency,''),COALESCE(status,''),
		       CASE WHEN renews_at IS NULL THEN NULL ELSE to_char(renews_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN trial_ends_at IS NULL THEN NULL ELSE to_char(trial_ends_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       COALESCE(auto_renew,false),
		       last_amount IS NOT NULL,COALESCE(last_amount::text,''),COALESCE(last_currency,''),
		       CASE WHEN last_paid_at IS NULL THEN NULL ELSE to_char(last_paid_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       standing,
		       count(*) OVER()
		FROM base
		WHERE ($2='' OR trade_name ILIKE '%'||$2||'%' OR legal_name ILIKE '%'||$2||'%' OR tax_id ILIKE '%'||$2||'%')
		  AND ($3='' OR plan_id::text=$3)
		  AND ($4='' OR standing=$4)
		ORDER BY CASE standing WHEN 'overdue' THEN 0 WHEN 'due_soon' THEN 1 WHEN 'trial' THEN 2 WHEN 'up_to_date' THEN 3 WHEN 'none' THEN 4 ELSE 5 END,
		         renews_at NULLS LAST,trade_name
		LIMIT $5 OFFSET $6
	`, subscriptionDueSoonDays, q, planID, standing, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "organizations_unavailable", "No pudimos cargar las empresas.")
		return
	}
	defer rows.Close()
	items := []platformOrganizationView{}
	total := 0
	for rows.Next() {
		var item platformOrganizationView
		var sub platformOrganizationSubscription
		var payment platformOrganizationPayment
		var hasSubscription, hasPayment bool
		if err = rows.Scan(
			&item.ID, &item.LegalName, &item.TradeName, &item.TaxID, &item.Active, &item.CreatedAt,
			&hasSubscription, &sub.Plan.ID, &sub.Plan.Code, &sub.Plan.Name,
			&sub.BillingCycle, &sub.PriceAmount, &sub.Currency, &sub.Status,
			&sub.RenewsAt, &sub.TrialEndsAt, &sub.AutoRenew,
			&hasPayment, &payment.Amount, &payment.Currency, &payment.PaidAt,
			&item.PaymentStanding, &total,
		); err != nil {
			fail(w, 503, "organizations_unavailable", "No pudimos cargar las empresas.")
			return
		}
		if hasSubscription {
			item.Subscription = &sub
		}
		if hasPayment {
			item.LastPayment = &payment
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		fail(w, 503, "organizations_unavailable", "No pudimos cargar las empresas.")
		return
	}

	planOptions := []optionView{}
	planRows, err := a.db.Query(r.Context(), `SELECT id::text,name FROM subscription_plans ORDER BY active DESC,monthly_price,name`)
	if err == nil {
		defer planRows.Close()
		for planRows.Next() {
			var option optionView
			if planRows.Scan(&option.Value, &option.Label) == nil {
				planOptions = append(planOptions, option)
			}
		}
	}

	writeJSON(w, 200, map[string]any{
		"items":           items,
		"total":           total,
		"page":            page,
		"pageSize":        size,
		"summary":         summary,
		"dueSoonDays":     subscriptionDueSoonDays,
		"planOptions":     planOptions,
		"standingOptions": paymentStandingOptions,
	})
}
