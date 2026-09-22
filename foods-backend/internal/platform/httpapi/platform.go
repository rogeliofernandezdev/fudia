package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type tenantOnboardingInput struct {
	LegalName        string   `json:"legalName"`
	TradeName        string   `json:"tradeName"`
	TaxID            string   `json:"taxId"`
	Timezone         string   `json:"timezone"`
	Country          string   `json:"country"`
	Currency         string   `json:"currency"`
	CurrencyPosition string   `json:"currencyPosition"`
	TaxName          string   `json:"taxName"`
	TaxRate          string   `json:"taxRate"`
	TaxIncluded      bool     `json:"taxIncluded"`
	LocationName     string   `json:"locationName"`
	LocationCode     string   `json:"locationCode"`
	Address          string   `json:"address"`
	LocationPhone    string   `json:"locationPhone"`
	LocationHours    string   `json:"locationHours"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
	AdminName        string   `json:"adminName"`
	AdminEmail       string   `json:"adminEmail"`
	AdminPassword    string   `json:"adminPassword"`
	PlanID           string   `json:"planId"`
	BillingCycle     string   `json:"billingCycle"`
	TermsAccepted    bool     `json:"termsAccepted"`
}

func (a *API) requirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := r.Context().Value(scopeKey{}).(scope)
		var allowed bool
		if err := a.db.QueryRow(r.Context(), `SELECT platform_admin FROM users WHERE id=$1 AND active`, s.UserID).Scan(&allowed); err != nil || !allowed {
			fail(w, 403, "platform_forbidden", "Solo el administrador de plataforma puede realizar esta acción.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) onboardTenant(w http.ResponseWriter, r *http.Request) {
	actor := r.Context().Value(scopeKey{}).(scope)
	var in tenantOnboardingInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	in.LegalName, in.TradeName, in.TaxID, in.Timezone = strings.TrimSpace(in.LegalName), strings.TrimSpace(in.TradeName), strings.TrimSpace(in.TaxID), strings.TrimSpace(in.Timezone)
	in.Country, in.Currency, in.CurrencyPosition = strings.ToUpper(strings.TrimSpace(in.Country)), strings.ToUpper(strings.TrimSpace(in.Currency)), strings.TrimSpace(in.CurrencyPosition)
	in.TaxName, in.LocationName, in.LocationCode, in.Address = strings.TrimSpace(in.TaxName), strings.TrimSpace(in.LocationName), strings.ToUpper(strings.TrimSpace(in.LocationCode)), strings.TrimSpace(in.Address)
	in.AdminName, in.AdminEmail = strings.TrimSpace(in.AdminName), strings.ToLower(strings.TrimSpace(in.AdminEmail))
	in.PlanID, in.BillingCycle = strings.TrimSpace(in.PlanID), strings.TrimSpace(in.BillingCycle)
	currency, fiscalOK := validateFiscalInput(&fiscalProfileInput{Country: in.Country, Currency: in.Currency, CurrencyPosition: in.CurrencyPosition, TaxName: in.TaxName, TaxRate: in.TaxRate, TaxIncluded: in.TaxIncluded, Default: true})
	if !fiscalOK || in.LegalName == "" || in.TradeName == "" || len(in.TaxID) < 6 || len(in.TaxID) > 32 || in.Timezone == "" || in.LocationName == "" || in.LocationCode == "" || in.AdminName == "" || !strings.Contains(in.AdminEmail, "@") || len(in.AdminPassword) < 8 || in.PlanID == "" || (in.BillingCycle != "monthly" && in.BillingCycle != "annual") || !in.TermsAccepted {
		fail(w, 400, "invalid_onboarding", "Completa correctamente empresa, configuración fiscal, local y administrador.")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "onboarding_unavailable", "No pudimos preparar el administrador.")
		return
	}
	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		fail(w, 503, "onboarding_unavailable", "No pudimos iniciar el registro de la empresa.")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var organizationID, profileID, locationID, userID, roleID string
	err = tx.QueryRow(r.Context(), `INSERT INTO organizations(legal_name,trade_name,tax_id,timezone) VALUES($1,$2,$3,$4) RETURNING id`, in.LegalName, in.TradeName, in.TaxID, in.Timezone).Scan(&organizationID)
	if err == nil {
		err = tx.QueryRow(r.Context(), `UPDATE organization_fiscal_profiles SET country_code=$1,currency=$2,currency_symbol=$3,currency_position=$4,currency_decimals=$5,tax_name=$6,tax_rate=$7,tax_included=$8,updated_at=now() WHERE organization_id=$9 AND is_default RETURNING id`, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, organizationID).Scan(&profileID)
	}
	if err == nil {
		err = tx.QueryRow(r.Context(), `INSERT INTO locations(organization_id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, organizationID, in.LocationName, in.LocationCode, in.Address, in.LocationPhone, in.LocationHours, in.Latitude, in.Longitude, in.Timezone, profileID).Scan(&locationID)
	}
	if err == nil {
		err = tx.QueryRow(r.Context(), `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,$4) RETURNING id`, organizationID, in.AdminEmail, in.AdminName, string(hash)).Scan(&userID)
	}
	if err == nil {
		roleID, err = seedOrganizationRoles(r.Context(), tx, organizationID)
	}
	if err == nil {
		err = createOrganizationSubscription(r.Context(), tx, organizationID, userID, in.PlanID, in.BillingCycle, in.TermsAccepted)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, userID, roleID, locationID)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'organization.onboarded','organization',$1,jsonb_build_object('platformActor',$4::text))`, organizationID, locationID, userID, actor.UserID)
	}
	if err != nil || tx.Commit(r.Context()) != nil {
		fail(w, 409, "onboarding_conflict", "No pudimos registrar la empresa. Verifica identificación fiscal y correo.")
		return
	}
	writeJSON(w, 201, map[string]string{"organizationId": organizationID, "fiscalProfileId": profileID, "locationId": locationID, "administratorId": userID})
}
