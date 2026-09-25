package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const tenantOnboardingFlow = "platform.onboarding"

const (
	onboardingStepValidate             = "validate_input"
	onboardingStepHashPassword         = "hash_admin_password"
	onboardingStepBeginTransaction     = "begin_transaction"
	onboardingStepCreateOrganization   = "create_organization"
	onboardingStepConfigureFiscal      = "configure_fiscal_profile"
	onboardingStepCreateLocation       = "create_location"
	onboardingStepCreateAdministrator  = "create_administrator"
	onboardingStepSeedRoles            = "seed_roles"
	onboardingStepSeedPaymentMethods   = "seed_payment_methods"
	onboardingStepSeedDefaults         = "seed_operational_defaults"
	onboardingStepCreateSubscription   = "create_subscription"
	onboardingStepAssignAdministrator  = "assign_administrator_role"
	onboardingStepWriteAudit           = "write_audit"
	onboardingStepCommitTransaction    = "commit_transaction"
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

func normalizeTenantOnboardingInput(in *tenantOnboardingInput) {
	in.LegalName = strings.TrimSpace(in.LegalName)
	in.TradeName = strings.TrimSpace(in.TradeName)
	in.TaxID = strings.TrimSpace(in.TaxID)
	in.Timezone = strings.TrimSpace(in.Timezone)
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	in.LocationName = strings.TrimSpace(in.LocationName)
	in.LocationCode = strings.ToUpper(strings.TrimSpace(in.LocationCode))
	in.Address = strings.TrimSpace(in.Address)
	in.LocationPhone = strings.TrimSpace(in.LocationPhone)
	in.LocationHours = strings.TrimSpace(in.LocationHours)
	in.AdminName = strings.TrimSpace(in.AdminName)
	in.AdminEmail = strings.ToLower(strings.TrimSpace(in.AdminEmail))
	in.PlanID = strings.TrimSpace(in.PlanID)
	in.BillingCycle = strings.TrimSpace(in.BillingCycle)
}

func validOnboardingEmail(value string) bool {
	if value == "" || len(value) > 180 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func validOnboardingCoordinates(latitude, longitude *float64) bool {
	if (latitude == nil) != (longitude == nil) {
		return false
	}
	if latitude == nil {
		return true
	}
	return *latitude >= -90 && *latitude <= 90 && *longitude >= -180 && *longitude <= 180
}

func rejectOnboarding(w http.ResponseWriter, r *http.Request, code, message string) {
	logFlowUserError(w, r, tenantOnboardingFlow, onboardingStepValidate, code)
	fail(w, http.StatusBadRequest, code, message)
}

func classifyOnboardingUserError(step string, err error) (status int, code, message string, ok bool) {
	if step == onboardingStepCreateSubscription && errors.Is(err, pgx.ErrNoRows) {
		return http.StatusConflict, "plan_unavailable", "El plan seleccionado ya no está disponible. Actualiza la página y selecciona un plan vigente.", true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "organizations_tax_id_key":
			return http.StatusConflict, "tax_id_already_registered", "La identificación fiscal ya está registrada en otra empresa.", true
		case "users_organization_id_email_key":
			return http.StatusConflict, "admin_email_already_registered", "El correo del administrador ya está registrado en esta empresa.", true
		case "locations_organization_id_code_key":
			return http.StatusConflict, "location_code_already_registered", "El código del local ya está registrado en esta empresa.", true
		}
		if step == onboardingStepCreateSubscription && pgErr.Code == "22P02" {
			return http.StatusBadRequest, "invalid_plan", "El plan seleccionado no es válido. Actualiza la página y vuelve a seleccionarlo.", true
		}
	}

	if step == onboardingStepCreateSubscription {
		switch err.Error() {
		case "invalid billing cycle":
			return http.StatusBadRequest, "invalid_billing_cycle", "El ciclo de facturación debe ser mensual o anual.", true
		case "terms not accepted":
			return http.StatusBadRequest, "terms_not_accepted", "Debes aceptar las condiciones del plan antes de registrar la empresa.", true
		}
	}

	return 0, "", "", false
}

func transientOnboardingError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if strings.HasPrefix(pgErr.Code, "08") || strings.HasPrefix(pgErr.Code, "53") {
		return true
	}
	switch pgErr.Code {
	case "40001", "40P01", "57P01", "57P02", "57P03":
		return true
	default:
		return false
	}
}

func failOnboardingStep(w http.ResponseWriter, r *http.Request, step string, err error) {
	if status, code, message, ok := classifyOnboardingUserError(step, err); ok {
		logFlowUserError(w, r, tenantOnboardingFlow, step, code)
		fail(w, status, code, message)
		return
	}

	status := http.StatusInternalServerError
	code := "onboarding_failed"
	message := "Ocurrió un error interno al registrar la empresa. Usa el código de seguimiento para solicitar revisión."
	if transientOnboardingError(err) {
		status = http.StatusServiceUnavailable
		code = "onboarding_unavailable"
		message = "No pudimos completar el registro de la empresa en este momento. Intenta nuevamente."
	}
	logFlowFailure(w, r, tenantOnboardingFlow, step, err)
	fail(w, status, code, message)
}

func (a *API) onboardTenant(w http.ResponseWriter, r *http.Request) {
	actor := r.Context().Value(scopeKey{}).(scope)
	logFlowStarted(w, r, tenantOnboardingFlow, "actor_user_id", actor.UserID)

	var in tenantOnboardingInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		rejectOnboarding(w, r, "invalid_request", "No pudimos leer los datos enviados. Revisa el formulario e inténtalo nuevamente.")
		return
	}
	normalizeTenantOnboardingInput(&in)

	if in.LegalName == "" || in.TradeName == "" || len(in.LegalName) > 180 || len(in.TradeName) > 180 {
		rejectOnboarding(w, r, "invalid_company_data", "Ingresa una razón social y un nombre comercial válidos.")
		return
	}
	if len(in.TaxID) < 6 || len(in.TaxID) > 32 {
		rejectOnboarding(w, r, "invalid_tax_id", "La identificación fiscal debe tener entre 6 y 32 caracteres.")
		return
	}
	if in.Timezone == "" {
		rejectOnboarding(w, r, "invalid_timezone", "Selecciona una zona horaria válida.")
		return
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		rejectOnboarding(w, r, "invalid_timezone", "La zona horaria seleccionada no es válida.")
		return
	}

	currency, fiscalOK := validateFiscalInput(&fiscalProfileInput{
		Country: in.Country, Currency: in.Currency, CurrencyPosition: in.CurrencyPosition,
		TaxName: in.TaxName, TaxRate: in.TaxRate, TaxIncluded: in.TaxIncluded, Default: true,
	})
	if !fiscalOK {
		rejectOnboarding(w, r, "invalid_fiscal_profile", "Revisa el país, la moneda, el impuesto y la posición del símbolo.")
		return
	}
	if in.LocationName == "" || in.LocationCode == "" || len(in.LocationName) > 180 || len(in.LocationCode) > 80 {
		rejectOnboarding(w, r, "invalid_location", "Ingresa un nombre y un código válidos para el primer local.")
		return
	}
	if !validOnboardingCoordinates(in.Latitude, in.Longitude) {
		rejectOnboarding(w, r, "invalid_location_coordinates", "Las coordenadas del local no son válidas.")
		return
	}
	if in.AdminName == "" || len(in.AdminName) > 180 {
		rejectOnboarding(w, r, "invalid_admin_name", "Ingresa el nombre completo del administrador.")
		return
	}
	if !validOnboardingEmail(in.AdminEmail) {
		rejectOnboarding(w, r, "invalid_admin_email", "Ingresa un correo electrónico válido para el administrador.")
		return
	}
	if len(in.AdminPassword) < 8 {
		rejectOnboarding(w, r, "invalid_admin_password", "La contraseña del administrador debe tener al menos 8 caracteres.")
		return
	}
	if len([]byte(in.AdminPassword)) > 72 {
		rejectOnboarding(w, r, "invalid_admin_password", "La contraseña del administrador no puede superar 72 bytes.")
		return
	}
	if in.PlanID == "" {
		rejectOnboarding(w, r, "invalid_plan", "Selecciona un plan comercial.")
		return
	}
	if in.BillingCycle != "monthly" && in.BillingCycle != "annual" {
		rejectOnboarding(w, r, "invalid_billing_cycle", "El ciclo de facturación debe ser mensual o anual.")
		return
	}
	if !in.TermsAccepted {
		rejectOnboarding(w, r, "terms_not_accepted", "Debes aceptar las condiciones del plan antes de registrar la empresa.")
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepValidate)

	hash, err := bcrypt.GenerateFromPassword([]byte(in.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		failOnboardingStep(w, r, onboardingStepHashPassword, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepHashPassword)

	tx, err := a.db.BeginTx(r.Context(), pgx.TxOptions{})
	if err != nil {
		failOnboardingStep(w, r, onboardingStepBeginTransaction, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepBeginTransaction)

	var organizationID, profileID, locationID, userID, roleID string

	err = tx.QueryRow(r.Context(), `INSERT INTO organizations(legal_name,trade_name,tax_id,timezone) VALUES($1,$2,$3,$4) RETURNING id`, in.LegalName, in.TradeName, in.TaxID, in.Timezone).Scan(&organizationID)
	if err != nil {
		failOnboardingStep(w, r, onboardingStepCreateOrganization, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepCreateOrganization, "organization_id", organizationID)

	err = tx.QueryRow(r.Context(), `UPDATE organization_fiscal_profiles SET country_code=$1,currency=$2,currency_symbol=$3,currency_position=$4,currency_decimals=$5,tax_name=$6,tax_rate=$7,tax_included=$8,updated_at=now() WHERE organization_id=$9 AND is_default RETURNING id`, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, organizationID).Scan(&profileID)
	if err != nil {
		failOnboardingStep(w, r, onboardingStepConfigureFiscal, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepConfigureFiscal, "organization_id", organizationID, "fiscal_profile_id", profileID)

	err = tx.QueryRow(r.Context(), `INSERT INTO locations(organization_id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, organizationID, in.LocationName, in.LocationCode, in.Address, in.LocationPhone, in.LocationHours, in.Latitude, in.Longitude, in.Timezone, profileID).Scan(&locationID)
	if err != nil {
		failOnboardingStep(w, r, onboardingStepCreateLocation, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepCreateLocation, "organization_id", organizationID, "location_id", locationID)

	err = tx.QueryRow(r.Context(), `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,$4) RETURNING id`, organizationID, in.AdminEmail, in.AdminName, string(hash)).Scan(&userID)
	if err != nil {
		failOnboardingStep(w, r, onboardingStepCreateAdministrator, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepCreateAdministrator, "organization_id", organizationID, "administrator_id", userID)

	roleID, err = seedOrganizationRoles(r.Context(), tx, organizationID)
	if err != nil {
		failOnboardingStep(w, r, onboardingStepSeedRoles, err)
		return
	}
	if roleID == "" {
		failOnboardingStep(w, r, onboardingStepSeedRoles, errors.New("administrator role seed returned empty id"))
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepSeedRoles, "organization_id", organizationID, "administrator_role_id", roleID)

	if err = seedOrganizationPaymentMethods(r.Context(), tx, organizationID); err != nil {
		failOnboardingStep(w, r, onboardingStepSeedPaymentMethods, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepSeedPaymentMethods, "organization_id", organizationID)

	if err = seedOrganizationOperationalDefaults(r.Context(), tx, organizationID, locationID, userID); err != nil {
		failOnboardingStep(w, r, onboardingStepSeedDefaults, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepSeedDefaults, "organization_id", organizationID, "location_id", locationID)

	if err = createOrganizationSubscription(r.Context(), tx, organizationID, userID, in.PlanID, in.BillingCycle, in.TermsAccepted); err != nil {
		failOnboardingStep(w, r, onboardingStepCreateSubscription, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepCreateSubscription, "organization_id", organizationID)

	if _, err = tx.Exec(r.Context(), `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, userID, roleID, locationID); err != nil {
		failOnboardingStep(w, r, onboardingStepAssignAdministrator, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepAssignAdministrator, "organization_id", organizationID, "administrator_id", userID)

	if _, err = tx.Exec(r.Context(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'organization.onboarded','organization',$1,jsonb_build_object('platformActor',$4::text))`, organizationID, locationID, userID, actor.UserID); err != nil {
		failOnboardingStep(w, r, onboardingStepWriteAudit, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepWriteAudit, "organization_id", organizationID)

	if err = tx.Commit(r.Context()); err != nil {
		failOnboardingStep(w, r, onboardingStepCommitTransaction, err)
		return
	}
	logFlowStep(w, r, tenantOnboardingFlow, onboardingStepCommitTransaction, "organization_id", organizationID)
	logFlowCompleted(w, r, tenantOnboardingFlow, "organization_id", organizationID, "location_id", locationID, "administrator_id", userID)

	writeJSON(w, http.StatusCreated, map[string]string{
		"organizationId": organizationID,
		"fiscalProfileId": profileID,
		"locationId": locationID,
		"administratorId": userID,
	})
}
