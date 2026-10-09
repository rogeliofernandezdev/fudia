package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type platformOrganizationUsage struct {
	Locations    int  `json:"locations"`
	Users        int  `json:"users"`
	MaxLocations *int `json:"maxLocations"`
	MaxUsers     *int `json:"maxUsers"`
}

type platformOrganizationDetailSubscription struct {
	PlanName        string  `json:"planName"`
	BillingCycle    string  `json:"billingCycle"`
	PriceAmount     string  `json:"priceAmount"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"startedAt"`
	RenewsAt        *string `json:"renewsAt"`
	TrialEndsAt     *string `json:"trialEndsAt"`
	AutoRenew       bool    `json:"autoRenew"`
	TermsVersion    *string `json:"termsVersion"`
	TermsAcceptedAt *string `json:"termsAcceptedAt"`
	ModuleCount     int     `json:"moduleCount"`
}

type platformOrganizationPayments struct {
	PaidCount  int     `json:"paidCount"`
	TotalPaid  string  `json:"totalPaid"`
	LastPaidAt *string `json:"lastPaidAt"`
}

type platformOrganizationAdmin struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Active bool   `json:"active"`
}

type platformOrganizationLocation struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Active  bool   `json:"active"`
}

type platformOrganizationDetail struct {
	ID              string                                  `json:"id"`
	LegalName       string                                  `json:"legalName"`
	TradeName       string                                  `json:"tradeName"`
	TaxID           string                                  `json:"taxId"`
	CountryCode     string                                  `json:"countryCode"`
	Currency        string                                  `json:"currency"`
	Timezone        string                                  `json:"timezone"`
	TaxName         string                                  `json:"taxName"`
	TaxRate         string                                  `json:"taxRate"`
	TaxIncluded     bool                                    `json:"taxIncluded"`
	Active          bool                                    `json:"active"`
	CreatedAt       string                                  `json:"createdAt"`
	PaymentStanding string                                  `json:"paymentStanding"`
	Usage           platformOrganizationUsage               `json:"usage"`
	Subscription    *platformOrganizationDetailSubscription `json:"subscription"`
	Payments        platformOrganizationPayments            `json:"payments"`
	Administrators  []platformOrganizationAdmin             `json:"administrators"`
	Locations       []platformOrganizationLocation          `json:"locations"`
}

const utcTimestampFormat = `'YYYY-MM-DD"T"HH24:MI:SS"Z"'`

// getPlatformOrganization devuelve la ficha completa de una empresa (solo platform admin).
func (a *API) getPlatformOrganization(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if !organizationIDPattern.MatchString(id) {
		fail(w, 404, "organization_not_found", "La empresa no existe.")
		return
	}
	unavailable := func(message string) { fail(w, 503, "organization_unavailable", message) }

	var out platformOrganizationDetail
	err := a.db.QueryRow(r.Context(), `
		SELECT o.id,o.legal_name,o.trade_name,COALESCE(o.tax_id,''),o.country_code,o.currency,o.timezone,
		       o.tax_name,(o.tax_rate*100)::numeric(7,2)::text,o.tax_included,o.active,
		       to_char(o.created_at AT TIME ZONE 'UTC',`+utcTimestampFormat+`),
		       (SELECT count(*) FROM locations l WHERE l.organization_id=o.id AND l.active),
		       (SELECT count(*) FROM users u WHERE u.organization_id=o.id AND u.active AND NOT u.platform_admin)
		FROM organizations o
		WHERE o.id=$1
	`, id).Scan(&out.ID, &out.LegalName, &out.TradeName, &out.TaxID, &out.CountryCode, &out.Currency, &out.Timezone,
		&out.TaxName, &out.TaxRate, &out.TaxIncluded, &out.Active, &out.CreatedAt, &out.Usage.Locations, &out.Usage.Users)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "organization_not_found", "La empresa no existe.")
		return
	}
	if err != nil {
		unavailable("No pudimos cargar la empresa.")
		return
	}

	if err = a.db.QueryRow(r.Context(), platformOrganizationsBase+` SELECT standing FROM base WHERE id=$2`, subscriptionDueSoonDays, id).Scan(&out.PaymentStanding); err != nil {
		unavailable("No pudimos cargar la empresa.")
		return
	}

	var sub platformOrganizationDetailSubscription
	err = a.db.QueryRow(r.Context(), `
		SELECT p.name,s.billing_cycle,s.price_amount::text,s.currency,s.status,
		       to_char(s.started_at AT TIME ZONE 'UTC',`+utcTimestampFormat+`),
		       CASE WHEN s.renews_at IS NULL THEN NULL ELSE to_char(s.renews_at AT TIME ZONE 'UTC',`+utcTimestampFormat+`) END,
		       CASE WHEN s.trial_ends_at IS NULL THEN NULL ELSE to_char(s.trial_ends_at AT TIME ZONE 'UTC',`+utcTimestampFormat+`) END,
		       s.auto_renew,s.terms_version,
		       CASE WHEN s.terms_accepted_at IS NULL THEN NULL ELSE to_char(s.terms_accepted_at AT TIME ZONE 'UTC',`+utcTimestampFormat+`) END,
		       cardinality(p.module_keys),p.max_locations,p.max_users
		FROM organization_subscriptions s
		JOIN subscription_plans p ON p.id=s.plan_id
		WHERE s.organization_id=$1
	`, id).Scan(&sub.PlanName, &sub.BillingCycle, &sub.PriceAmount, &sub.Currency, &sub.Status, &sub.StartedAt,
		&sub.RenewsAt, &sub.TrialEndsAt, &sub.AutoRenew, &sub.TermsVersion, &sub.TermsAcceptedAt, &sub.ModuleCount,
		&out.Usage.MaxLocations, &out.Usage.MaxUsers)
	switch {
	case err == nil:
		out.Subscription = &sub
	case !errors.Is(err, pgx.ErrNoRows):
		unavailable("No pudimos cargar la suscripción de la empresa.")
		return
	}

	if err = a.db.QueryRow(r.Context(), `
		SELECT count(*),COALESCE(sum(amount),0)::numeric(12,2)::text,
		       CASE WHEN max(paid_at) IS NULL THEN NULL ELSE to_char(max(paid_at) AT TIME ZONE 'UTC',`+utcTimestampFormat+`) END
		FROM subscription_payments
		WHERE organization_id=$1 AND status='paid'
	`, id).Scan(&out.Payments.PaidCount, &out.Payments.TotalPaid, &out.Payments.LastPaidAt); err != nil {
		unavailable("No pudimos cargar los pagos de la empresa.")
		return
	}

	out.Administrators = []platformOrganizationAdmin{}
	adminRows, err := a.db.Query(r.Context(), `
		SELECT DISTINCT u.full_name,u.email,u.active
		FROM users u
		JOIN user_roles ur ON ur.user_id=u.id
		JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=u.organization_id
		WHERE u.organization_id=$1 AND NOT u.platform_admin AND ro.system_key='administrator'
		ORDER BY u.active DESC,u.full_name
	`, id)
	if err != nil {
		unavailable("No pudimos cargar los administradores.")
		return
	}
	for adminRows.Next() {
		var item platformOrganizationAdmin
		if err = adminRows.Scan(&item.Name, &item.Email, &item.Active); err != nil {
			adminRows.Close()
			unavailable("No pudimos cargar los administradores.")
			return
		}
		out.Administrators = append(out.Administrators, item)
	}
	adminRows.Close()

	out.Locations = []platformOrganizationLocation{}
	locationRows, err := a.db.Query(r.Context(), `
		SELECT id,name,code,address,phone,active
		FROM locations
		WHERE organization_id=$1
		ORDER BY active DESC,created_at
	`, id)
	if err != nil {
		unavailable("No pudimos cargar los locales.")
		return
	}
	defer locationRows.Close()
	for locationRows.Next() {
		var item platformOrganizationLocation
		if err = locationRows.Scan(&item.ID, &item.Name, &item.Code, &item.Address, &item.Phone, &item.Active); err != nil {
			unavailable("No pudimos cargar los locales.")
			return
		}
		out.Locations = append(out.Locations, item)
	}
	writeJSON(w, 200, out)
}
