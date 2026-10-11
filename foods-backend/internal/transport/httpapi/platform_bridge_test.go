package httpapi

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
