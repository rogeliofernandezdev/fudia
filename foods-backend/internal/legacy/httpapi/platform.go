package httpapi

import (
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type tenantOnboardingInput = platformdomain.TenantOnboardingInput

var normalizeTenantOnboardingInput = platformdomain.NormalizeTenantOnboardingInput
var validOnboardingEmail = platformdomain.ValidOnboardingEmail
var validOnboardingCoordinates = platformdomain.ValidOnboardingCoordinates

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

func (a *API) requirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := r.Context().Value(scopeKey{}).(scope)
		if err := a.handlers.Platform.Authorize(r.Context(), s.UserID); err != nil {
			shared.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *API) onboardTenant(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.OnboardTenant(w, r)
}
