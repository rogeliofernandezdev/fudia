package application

import (
	"context"
	"errors"
	identitycontracts "github.com/foods-platform/foods-backend/internal/modules/identity/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"time"
)

type FiscalCatalog interface {
	ValidateFiscal(context.Context, *orgcontracts.FiscalProfileInput, bool) (orgcontracts.CurrencyOption, bool, error)
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

// TenantCreateStep identifica cada sentencia del envío de CreateTenant.
type TenantCreateStep int

const (
	TenantStepOrganization TenantCreateStep = iota
	TenantStepFiscal
	TenantStepLocation
	TenantStepAdministrator
)

var tenantCreateSteps = map[TenantCreateStep]string{
	TenantStepOrganization:  onboardingStepCreateOrganization,
	TenantStepFiscal:        onboardingStepConfigureFiscal,
	TenantStepLocation:      onboardingStepCreateLocation,
	TenantStepAdministrator: onboardingStepCreateAdministrator,
}

// TenantCreateError conserva el paso que falló dentro del envío, para que el
// flujo registre y clasifique el error igual que cuando eran consultas separadas.
type TenantCreateError struct {
	Step TenantCreateStep
	Err  error
}

func (e *TenantCreateError) Error() string { return e.Err.Error() }
func (e *TenantCreateError) Unwrap() error { return e.Err }

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

	currency, fiscalOK, catalogErr := a.catalogs.ValidateFiscal(ctx, &orgcontracts.FiscalProfileInput{
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

	var roleID string
	tenant, err := a.repo.CreateTenant(tx.Context(), s, in, currency, hash)
	if err != nil {
		step := onboardingStepCreateOrganization
		var stepErr *TenantCreateError
		if errors.As(err, &stepErr) {
			step = tenantCreateSteps[stepErr.Step]
		}
		return result, a.onboardingFailure(ctx, step, err)
	}
	organizationID, profileID, locationID, userID := tenant.OrganizationID, tenant.FiscalProfileID, tenant.LocationID, tenant.AdministratorID
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCreateOrganization, nil, "organization_id", organizationID)
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepConfigureFiscal, nil, "organization_id", organizationID, "fiscal_profile_id", profileID)
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepCreateLocation, nil, "organization_id", organizationID, "location_id", locationID)
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

	// Categoría, mesas, modalidad de atención y plantillas operativas comparten
	// la transacción: si falla cualquier paso posterior, nada queda creado.
	defaults := domain.NewTenantOperationalDefaults()
	if err = defaults.Validate(); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepSeedDefaults, err)
	}
	if err = a.repo.SeedOperationalDefaults(tx.Context(), organizationID, locationID, userID, defaults); err != nil {
		return result, a.onboardingFailure(ctx, onboardingStepSeedDefaults, err)
	}
	a.flow.Log(ctx, "application_flow_step_completed", onboardingStepSeedDefaults, nil, "organization_id", organizationID, "location_id", locationID)

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
	seeds := identitycontracts.DefaultOrganizationRoles
	ids, err := a.repo.SeedRoles(ctx, org, seeds)
	if err != nil {
		return "", err
	}
	if len(ids) != len(seeds) {
		return "", errors.New("role seed returned an unexpected number of ids")
	}
	for i, seed := range seeds {
		if seed.SystemKey == "administrator" {
			return ids[i], nil
		}
	}
	return "", nil
}
