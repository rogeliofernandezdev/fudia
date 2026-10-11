package application

import (
	"context"
	"errors"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	orgdomain "github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"time"
)

type FiscalCatalog interface {
	ValidateFiscal(context.Context, *orgdomain.FiscalProfileInput, bool) (orgdomain.CurrencyOption, bool, error)
}
type SubscriptionCreator interface {
	Create(context.Context, string, string, string, string, bool) error
}
type PasswordHasher interface{ Hash(string) (string, error) }
type FailureCategory string

const (
	FailureUnknown      FailureCategory = "unknown"
	FailureMissing      FailureCategory = "missing"
	FailureTaxID        FailureCategory = "tax_id"
	FailureEmail        FailureCategory = "email"
	FailureLocationCode FailureCategory = "location_code"
	FailureIdentifier   FailureCategory = "identifier"
	FailureTransient    FailureCategory = "transient"
)

type FailureClassifier interface{ Classify(error) FailureCategory }
type FlowObserver interface {
	Log(context.Context, string, string, error, ...any)
}

const (
	onboardingStepValidate            = "validate_input"
	onboardingStepHashPassword        = "hash_admin_password"
	onboardingStepBeginTransaction    = "begin_transaction"
	onboardingStepCreateOrganization  = "create_organization"
	onboardingStepConfigureFiscal     = "configure_fiscal_profile"
	onboardingStepCreateLocation      = "create_location"
	onboardingStepCreateAdministrator = "create_administrator"
	onboardingStepSeedRoles           = "seed_roles"
	onboardingStepSeedPaymentMethods  = "seed_payment_methods"
	onboardingStepSeedDefaults        = "seed_operational_defaults"
	onboardingStepCreateSubscription  = "create_subscription"
	onboardingStepAssignAdministrator = "assign_administrator_role"
	onboardingStepWriteAudit          = "write_audit"
	onboardingStepCommitTransaction   = "commit_transaction"
)
const tenantOnboardingFlow = "platform.onboarding"

func (a *Platform) reject(ctx context.Context, code, msg string) error {
	a.flow.Log(ctx, "application_flow_rejected", onboardingStepValidate, nil, "error_code", code, "error_kind", "user")
	return platformFailure(apperror.Invalid, code, msg)
}
func ClassifyOnboardingUserError(step string, category FailureCategory, err error) (*apperror.Error, bool) {
	if step == onboardingStepCreateSubscription && category == FailureMissing {
		return apperror.New(apperror.Conflict, "plan_unavailable", "El plan seleccionado ya no está disponible. Actualiza la página y selecciona un plan vigente."), true
	}
	switch category {
	case FailureTaxID:
		return apperror.New(apperror.Conflict, "tax_id_already_registered", "La identificación fiscal ya está registrada en otra empresa."), true
	case FailureEmail:
		return apperror.New(apperror.Conflict, "admin_email_already_registered", "El correo del administrador ya está registrado en esta empresa."), true
	case FailureLocationCode:
		return apperror.New(apperror.Conflict, "location_code_already_registered", "El código del local ya está registrado en esta empresa."), true
	}
	if step == onboardingStepCreateSubscription {
		if category == FailureIdentifier {
			return apperror.New(apperror.Invalid, "invalid_plan", "El plan seleccionado no es válido. Actualiza la página y vuelve a seleccionarlo."), true
		}
		if err != nil {
			switch err.Error() {
			case "invalid billing cycle":
				return apperror.New(apperror.Invalid, "invalid_billing_cycle", "El ciclo de facturación debe ser mensual o anual."), true
			case "terms not accepted":
				return apperror.New(apperror.Invalid, "terms_not_accepted", "Debes aceptar las condiciones del plan antes de registrar la empresa."), true
			}
		}
	}
	return nil, false
}
func (a *Platform) onboardingFailure(ctx context.Context, step string, err error) error {
	category := a.failures.Classify(err)
	if failure, ok := ClassifyOnboardingUserError(step, category, err); ok {
		a.flow.Log(ctx, "application_flow_rejected", step, nil, "error_code", failure.Code, "error_kind", "user")
		return failure
	}
	a.flow.Log(ctx, "application_flow_failed", step, err, "error_kind", "system")
	if category == FailureTransient {
		return platformFailure(apperror.Unavailable, "onboarding_unavailable", "No pudimos completar el registro de la empresa en este momento. Intenta nuevamente.")
	}
	return platformFailure(apperror.Internal, "onboarding_failed", "Ocurrió un error interno al registrar la empresa. Usa el código de seguimiento para solicitar revisión.")
}
func (a *Platform) Onboard(ctx context.Context, s scope, in domain.TenantOnboardingInput) (map[string]string, error) {
	var result map[string]string
	a.flow.Log(ctx, "application_flow_started", "", nil, "actor_user_id", s.UserID)
	domain.NormalizeTenantOnboardingInput(&in)

	if in.LegalName == "" || in.TradeName == "" || len(in.LegalName) > 180 || len(in.TradeName) > 180 {
		return result, a.reject(ctx, "invalid_company_data", "Ingresa una razón social y un nombre comercial válidos.")
	}
	if len(in.TaxID) < 6 || len(in.TaxID) > 32 {
		return result, a.reject(ctx, "invalid_tax_id", "La identificación fiscal debe tener entre 6 y 32 caracteres.")
	}
	if in.Timezone == "" {
		return result, a.reject(ctx, "invalid_timezone", "Selecciona una zona horaria válida.")
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return result, a.reject(ctx, "invalid_timezone", "La zona horaria seleccionada no es válida.")
	}

	currency, fiscalOK, catalogErr := a.catalogs.ValidateFiscal(ctx, &orgdomain.FiscalProfileInput{
		Country: in.Country, Currency: in.Currency, CurrencyPosition: in.CurrencyPosition,
		TaxName: in.TaxName, TaxRate: in.TaxRate, TaxIncluded: in.TaxIncluded, Default: true,
	}, true)
	if catalogErr != nil {
		a.flow.Log(ctx, "application_flow_failed", onboardingStepValidate, catalogErr)
		return result, platformFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos validar la configuración global de países, monedas y WhatsApp. Intenta nuevamente.")
	}
	if !fiscalOK {
		return result, a.reject(ctx, "invalid_fiscal_profile", "El país o su moneda predeterminada no están habilitados, o el país todavía no tiene un canal de WhatsApp operativo.")
	}
	if in.LocationName == "" || in.LocationCode == "" || len(in.LocationName) > 180 || len(in.LocationCode) > 80 {
		return result, a.reject(ctx, "invalid_location", "Ingresa un nombre y un código válidos para el primer local.")
	}
	if !domain.ValidOnboardingCoordinates(in.Latitude, in.Longitude) {
		return result, a.reject(ctx, "invalid_location_coordinates", "Las coordenadas del local no son válidas.")
	}
	if in.AdminName == "" || len(in.AdminName) > 180 {
		return result, a.reject(ctx, "invalid_admin_name", "Ingresa el nombre completo del administrador.")
	}
	if !domain.ValidOnboardingEmail(in.AdminEmail) {
		return result, a.reject(ctx, "invalid_admin_email", "Ingresa un correo electrónico válido para el administrador.")
	}
	if len(in.AdminPassword) < 8 {
		return result, a.reject(ctx, "invalid_admin_password", "La contraseña del administrador debe tener al menos 8 caracteres.")
	}
	if len([]byte(in.AdminPassword)) > 72 {
		return result, a.reject(ctx, "invalid_admin_password", "La contraseña del administrador no puede superar 72 bytes.")
	}
	if in.PlanID == "" {
		return result, a.reject(ctx, "invalid_plan", "Selecciona un plan comercial.")
	}
	if in.BillingCycle != "monthly" && in.BillingCycle != "annual" {
		return result, a.reject(ctx, "invalid_billing_cycle", "El ciclo de facturación debe ser mensual o anual.")
	}
	if !in.TermsAccepted {
		return result, a.reject(ctx, "terms_not_accepted", "Debes aceptar las condiciones del plan antes de registrar la empresa.")
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepValidate, nil)

	hash, err := a.passwords.Hash(in.AdminPassword)
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepHashPassword, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepHashPassword, nil)

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepBeginTransaction, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepBeginTransaction, nil)

	var organizationID, profileID, locationID, userID, roleID string

	err = func() error {
		var err error
		organizationID, err = a.repo.CreateOrganization(tx.Context(), s, in)
		return err
	}()
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepCreateOrganization, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCreateOrganization, nil, "organization_id", organizationID)

	err = func() error {
		var err error
		profileID, err = a.repo.ConfigureFiscal(tx.Context(), s, organizationID, in, currency)
		return err
	}()
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepConfigureFiscal, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepConfigureFiscal, nil, "organization_id", organizationID, "fiscal_profile_id", profileID)

	err = func() error {
		var err error
		locationID, err = a.repo.CreateLocation(tx.Context(), s, organizationID, profileID, in)
		return err
	}()
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepCreateLocation, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCreateLocation, nil, "organization_id", organizationID, "location_id", locationID)

	err = func() error {
		var err error
		userID, err = a.repo.CreateAdministrator(tx.Context(), s, organizationID, in, hash)
		return err
	}()
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepCreateAdministrator, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCreateAdministrator, nil, "organization_id", organizationID, "administrator_id", userID)

	roleID, err = a.seedRoles(tx.Context(), organizationID)
	if err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepSeedRoles, err)
	}
	if roleID == "" {
		return result, a.onboardingFailure(ctx, onboardingStepSeedRoles, errors.New("administrator role seed returned empty id"))
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepSeedRoles, nil, "organization_id", organizationID, "administrator_role_id", roleID)

	if err = a.repo.SeedPaymentMethods(tx.Context(), organizationID); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepSeedPaymentMethods, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepSeedPaymentMethods, nil, "organization_id", organizationID)

	if err = a.repo.SeedOperationalDefaults(tx.Context(), organizationID, locationID, userID); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepSeedDefaults, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepSeedDefaults, nil, "organization_id", organizationID, "location_id", locationID)

	if _, err = a.repo.CreateOperationalSetup(tx.Context(), s, organizationID); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepSeedDefaults, err)
	}

	if err = a.subscriptions.Create(tx.Context(), organizationID, userID, in.PlanID, in.BillingCycle, in.TermsAccepted); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepCreateSubscription, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCreateSubscription, nil, "organization_id", organizationID)

	if _, err = a.repo.AssignAdministrator(tx.Context(), s, userID, roleID, locationID); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepAssignAdministrator, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepAssignAdministrator, nil, "organization_id", organizationID, "administrator_id", userID)

	if _, err = a.repo.AuditOnboarded(tx.Context(), s, organizationID, locationID, userID); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepWriteAudit, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepWriteAudit, nil, "organization_id", organizationID)

	if err = tx.Commit(ctx); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepCommitTransaction, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCommitTransaction, nil, "organization_id", organizationID)
	a.flow.Log(ctx, "application_flow_completed", "", nil, "organization_id", organizationID, "location_id", locationID, "administrator_id", userID)

	return map[string]string{
		"organizationId":  organizationID,
		"fiscalProfileId": profileID,
		"locationId":      locationID,
		"administratorId": userID,
	}, nil

}
func (a *Platform) seedRoles(ctx context.Context, org string) (string, error) {
	admin := ""
	for _, seed := range identitydomain.DefaultOrganizationRoles {
		id, err := a.repo.SeedRole(ctx, org, seed)
		if err != nil {
			return "", err
		}
		if seed.SystemKey == "administrator" {
			admin = id
		}
	}
	return admin, nil
}
