package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	subscriptioncontracts "github.com/foods-platform/foods-backend/internal/modules/subscriptions/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math/big"
	"regexp"
	"strings"
	"time"
)

// Alias preserves error identity for callers and PostgreSQL adapters.
var ErrNotFound = subscriptioncontracts.ErrNotFound

type scope = requestscope.Scope
type Subscriptions struct {
	repo  SubscriptionRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func New(repo SubscriptionRepository, work unitofwork.Manager, audit audit.Repository) *Subscriptions {
	return &Subscriptions{repo, work, audit}
}
func subscriptionFailure(k apperror.Kind, c, m string) error { return apperror.New(k, c, m) }

type SubscriptionRepository interface {
	Plan(context.Context, string, bool) (domain.SubscriptionPlanView, error)
	PlanOrganizations(context.Context, string) ([]string, error)
	SaveModule(context.Context, string, string, bool) error
	CreatePlan(ctx context.Context, s scope, in domain.SubscriptionPlanInput) (string, error)
	PlanLimitsExceeded(ctx context.Context, s scope, id string, in domain.SubscriptionPlanInput) (bool, error)
	UpdatePlan(ctx context.Context, s scope, id string, in domain.SubscriptionPlanInput) (int64, error)
	PlatformPlan(context.Context, string) (domain.PlatformPlanView, error)
	Plans(ctx context.Context) ([]domain.PlatformPlanView, error)
	AvailablePlans(ctx context.Context) ([]domain.SubscriptionPlanView, error)
	OrganizationExists(context.Context, string) (bool, error)
	MarkPastDue(ctx context.Context, s scope, organizationID string) (int64, error)
	Subscription(context.Context, string) (domain.OrganizationSubscriptionView, error)
	LockContract(ctx context.Context, s scope, organizationID string) (string, string, string, string, *string, *time.Time, *time.Time, *time.Time, *time.Time, *time.Time, *time.Time, error)
	ActiveLocations(ctx context.Context, s scope, organizationID string) (int, error)
	ActiveUsers(ctx context.Context, s scope, organizationID string) (int, error)
	UpdateContract(ctx context.Context, s scope, organizationID string, plan domain.SubscriptionPlanView, in domain.SubscriptionUpdateInput, price string, trialStarts *time.Time, trialEnds *time.Time, periodStarts *time.Time, periodEnds *time.Time, renews *time.Time, termsVersion *string, termsAcceptedAt *time.Time, termsAcceptedBy *string, cancelledAt *time.Time) (int64, error)
	LockPaymentContract(ctx context.Context, s scope, organizationID string) (string, string, string, []string, *time.Time, error)
	RecordPayment(ctx context.Context, s scope, organizationID string, subscriptionID string, in domain.SubscriptionPaymentInput, external *string, periodStart time.Time, periodEnd time.Time, paidAt *time.Time) (string, error)
	RenewContract(ctx context.Context, s scope, organizationID string, periodStart time.Time, periodEnd time.Time) (int64, error)
	CreateContract(ctx context.Context, organizationID string, administratorID string, plan domain.SubscriptionPlanView, billingCycle string, price string, status string, trialStarts *time.Time, trialEnds *time.Time, now time.Time, periodEnd time.Time) error
}

var planCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,39}$`)

func NormalizePlanInput(in *domain.SubscriptionPlanInput) bool {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.MonthlyPrice = strings.TrimSpace(in.MonthlyPrice)
	in.AnnualPrice = strings.TrimSpace(in.AnnualPrice)
	in.TermsVersion = strings.TrimSpace(in.TermsVersion)
	if !planCodePattern.MatchString(in.Code) || in.Name == "" || len(in.Name) > 120 || len(in.Description) > 500 || in.TermsVersion == "" || len(in.TermsVersion) > 80 {
		return false
	}
	if _, ok := orgcontracts.CurrencyByCode(in.Currency); !ok {
		return false
	}
	monthly, monthlyOK := new(big.Rat).SetString(in.MonthlyPrice)
	annual, annualOK := new(big.Rat).SetString(in.AnnualPrice)
	if !monthlyOK || !annualOK || monthly.Sign() < 0 || annual.Sign() < 0 || in.TrialDays < 0 || in.TrialDays > 365 {
		return false
	}
	if (in.MaxLocations != nil && *in.MaxLocations < 1) || (in.MaxUsers != nil && *in.MaxUsers < 1) || (in.MaxConciergeConversations != nil && *in.MaxConciergeConversations < 1) {
		return false
	}
	seen := map[string]bool{}
	modules := make([]string, 0, len(in.ModuleKeys))
	for _, raw := range in.ModuleKeys {
		key := strings.TrimSpace(raw)
		if key == "" || seen[key] || !domain.ModuleCanActivate(key) {
			return false
		}
		seen[key] = true
		modules = append(modules, key)
	}
	if len(modules) == 0 {
		return false
	}
	in.ModuleKeys = modules
	return true
}
func (a *Subscriptions) SyncModules(ctx context.Context, organizationID string, keys []string) error {
	allowed := map[string]bool{}
	for _, key := range keys {
		allowed[key] = true
	}
	for _, m := range domain.ModuleCatalog {
		if err := a.repo.SaveModule(ctx, organizationID, m.Key, allowed[m.Key] && domain.ModuleCanActivate(m.Key)); err != nil {
			return err
		}
	}
	return nil
}
func (a *Subscriptions) SavePlan(ctx context.Context, s scope, requestID string, in domain.SubscriptionPlanInput, create bool) (domain.PlatformPlanView, error) {
	var result domain.PlatformPlanView
	if !NormalizePlanInput(&in) {
		return result, subscriptionFailure(apperror.Invalid, "invalid_plan", "Revisa código, precios, límites, módulos y versión de condiciones.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "No pudimos guardar el plan.")
	}
	defer tx.Rollback(ctx)

	id := requestID
	if create {
		err = func() error { var err error; id, err = a.repo.CreatePlan(tx.Context(), s, in); return err }()
	} else {
		id = requestID
		var exceeds bool
		err = func() error {
			var err error
			exceeds, err = a.repo.PlanLimitsExceeded(tx.Context(), s, id, in)
			return err
		}()
		if err == nil && exceeds {
			return result, subscriptionFailure(apperror.Conflict, "plan_limit_conflict", "El nuevo límite es menor que el uso actual de una empresa suscrita.")
		}
		if err == nil {
			tag, updateErr := a.repo.UpdatePlan(tx.Context(), s, id, in)
			err = updateErr
			if err == nil && tag == 0 {
				err = ErrNotFound
			}
		}
	}
	if err != nil {
		return result, subscriptionFailure(apperror.Conflict, "plan_conflict", "El código ya existe o el plan no puede modificarse.")
	}

	if !create {
		orgIDs, queryErr := a.repo.PlanOrganizations(tx.Context(), id)
		if queryErr != nil {
			return result, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "No pudimos sincronizar el plan.")
		}
		for _, orgID := range orgIDs {
			if err = a.SyncModules(tx.Context(), orgID, in.ModuleKeys); err != nil {
				return result, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "No pudimos sincronizar los módulos del plan.")
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "No pudimos guardar el plan.")
	}
	_ = a.audit.Record(ctx, s, map[bool]string{true: "subscription_plan.created", false: "subscription_plan.updated"}[create], "subscription_plan", id)
	plan, err := a.repo.PlatformPlan(ctx, id)
	if err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "El plan se guardó pero no pudimos volver a cargarlo.")
	}
	return plan, nil

}
func (a *Subscriptions) Plans(ctx context.Context) ([]domain.PlatformPlanView, error) {
	out, err := a.repo.Plans(ctx)
	if err != nil {
		return nil, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "No pudimos cargar los planes.")
	}
	return out, nil
}
func (a *Subscriptions) AvailablePlans(ctx context.Context) ([]domain.SubscriptionPlanView, error) {
	out, err := a.repo.AvailablePlans(ctx)
	if err != nil {
		return nil, subscriptionFailure(apperror.Unavailable, "plans_unavailable", "No pudimos cargar los planes.")
	}
	return out, nil
}

var organizationIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (a *Subscriptions) organization(ctx context.Context, s scope, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return s.OrganizationID, nil
	}
	if organizationIDPattern.MatchString(id) {
		exists, err := a.repo.OrganizationExists(ctx, id)
		if err == nil && exists {
			return id, nil
		}
	}
	return "", subscriptionFailure(apperror.NotFound, "organization_not_found", "La empresa no existe.")
}
func (a *Subscriptions) Get(ctx context.Context, s scope, id string) (domain.OrganizationSubscriptionView, error) {
	var out domain.OrganizationSubscriptionView
	org, err := a.organization(ctx, s, id)
	if err != nil {
		return out, err
	}
	_, _ = a.repo.MarkPastDue(ctx, s, org)
	out, err = a.repo.Subscription(ctx, org)
	if errors.Is(err, ErrNotFound) {
		return out, subscriptionFailure(apperror.NotFound, "subscription_not_found", "La empresa todavía no tiene una suscripción registrada.")
	}
	if err != nil {
		return out, subscriptionFailure(apperror.Unavailable, "subscription_unavailable", "No pudimos cargar la suscripción.")
	}
	return out, nil
}
func (a *Subscriptions) Update(ctx context.Context, s scope, id string, in domain.SubscriptionUpdateInput) (domain.OrganizationSubscriptionView, error) {
	var result domain.OrganizationSubscriptionView
	organizationID, err := a.organization(ctx, s, id)
	if err != nil {
		return result, err
	}
	in.PlanID = strings.TrimSpace(in.PlanID)
	in.BillingCycle = strings.TrimSpace(in.BillingCycle)
	in.Status = strings.TrimSpace(in.Status)
	if in.PlanID == "" || (in.BillingCycle != "monthly" && in.BillingCycle != "annual") || (in.Status != "trial" && in.Status != "active" && in.Status != "past_due" && in.Status != "cancelled") {
		return result, subscriptionFailure(apperror.Invalid, "invalid_subscription", "Plan, ciclo y estado son obligatorios.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "subscription_unavailable", "No pudimos actualizar la suscripción.")
	}
	defer tx.Rollback(ctx)

	plan, err := a.repo.Plan(tx.Context(), in.PlanID, true)
	if err != nil {
		return result, subscriptionFailure(apperror.NotFound, "plan_not_found", "El plan seleccionado no está disponible.")
	}
	if in.Status == "trial" && plan.TrialDays == 0 {
		return result, subscriptionFailure(apperror.Conflict, "trial_unavailable", "El plan seleccionado no incluye prueba gratuita.")
	}
	var currentPlanID, currentCycle, currentStatus, currentPrice string
	var currentTerms *string
	var currentTrialStarts, currentTrialEnds, currentPeriodStarts, currentPeriodEnds, currentRenews, currentCancelledAt *time.Time
	if err = func() error {
		var err error
		currentPlanID, currentCycle, currentStatus, currentPrice, currentTerms, currentTrialStarts, currentTrialEnds, currentPeriodStarts, currentPeriodEnds, currentRenews, currentCancelledAt, err = a.repo.LockContract(tx.Context(), s, organizationID)
		return err
	}(); err != nil {
		return result, subscriptionFailure(apperror.NotFound, "subscription_not_found", "La empresa todavía no tiene una suscripción.")
	}
	var locationCount, userCount int
	_ = func() error {
		var err error
		locationCount, err = a.repo.ActiveLocations(tx.Context(), s, organizationID)
		return err
	}()
	_ = func() error {
		var err error
		userCount, err = a.repo.ActiveUsers(tx.Context(), s, organizationID)
		return err
	}()
	if (plan.MaxLocations != nil && locationCount > *plan.MaxLocations) || (plan.MaxUsers != nil && userCount > *plan.MaxUsers) {
		return result, subscriptionFailure(apperror.Conflict, "plan_limit_conflict", "La empresa supera los límites del plan seleccionado.")
	}
	planChanged := currentPlanID != plan.ID
	cycleChanged := currentCycle != in.BillingCycle
	needsTerms := planChanged || currentTerms == nil || *currentTerms != plan.TermsVersion
	if needsTerms && !in.TermsAccepted {
		return result, subscriptionFailure(apperror.Conflict, "terms_required", "La versión vigente de condiciones debe aceptarse antes de aplicar el cambio.")
	}

	now := time.Now().UTC()
	price := currentPrice
	if planChanged || cycleChanged {
		price = plan.MonthlyPrice
		if in.BillingCycle == "annual" {
			price = plan.AnnualPrice
		}
	}
	trialStarts, trialEnds := currentTrialStarts, currentTrialEnds
	periodStarts, periodEnds, renews := currentPeriodStarts, currentPeriodEnds, currentRenews
	cancelledAt := currentCancelledAt
	contractChanged := planChanged || cycleChanged || currentStatus != in.Status
	if contractChanged {
		switch in.Status {
		case "trial":
			start := now
			end := now.AddDate(0, 0, plan.TrialDays)
			trialStarts = &start
			trialEnds = &end
			periodStarts = &start
			periodEnds = &end
			renews = &end
		case "active":
			end := domain.SubscriptionPeriodEnd(now, in.BillingCycle)
			trialStarts = nil
			trialEnds = nil
			periodStarts = &now
			periodEnds = &end
			renews = &end
			cancelledAt = nil
		case "past_due":
			cancelledAt = nil
			if periodStarts == nil {
				periodStarts = &now
			}
			if periodEnds == nil {
				end := domain.SubscriptionPeriodEnd(now, in.BillingCycle)
				periodEnds = &end
				renews = &end
			}
		case "cancelled":
			renews = nil
			if currentStatus != "cancelled" {
				cancelledAt = &now
			}
		}
	}
	termsVersion := currentTerms
	var termsAcceptedAt *time.Time
	var termsAcceptedBy *string
	if in.TermsAccepted {
		termsVersion = &plan.TermsVersion
		termsAcceptedAt = &now
		termsAcceptedBy = &s.UserID
	}
	_, err = a.repo.UpdateContract(tx.Context(), s, organizationID, plan, in, price, trialStarts, trialEnds, periodStarts, periodEnds, renews, termsVersion, termsAcceptedAt, termsAcceptedBy, cancelledAt)
	if err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "subscription_unavailable", "No pudimos actualizar la suscripción.")
	}
	moduleKeys := plan.ModuleKeys
	if in.Status == "cancelled" {
		moduleKeys = []string{}
	}
	if err = a.SyncModules(tx.Context(), organizationID, moduleKeys); err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "subscription_unavailable", "No pudimos aplicar los módulos del plan.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "subscription_unavailable", "No pudimos confirmar el cambio de suscripción.")
	}
	_ = a.audit.Record(ctx, s, "subscription.updated", "organization", organizationID)
	return a.Get(ctx, s, organizationID)

}
func (a *Subscriptions) RecordPayment(ctx context.Context, s scope, requestID string, in domain.SubscriptionPaymentInput) (map[string]string, error) {
	var result map[string]string
	organizationID, err := a.organization(ctx, s, requestID)
	if err != nil {
		return result, err
	}
	in.Amount = strings.TrimSpace(in.Amount)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Status = strings.TrimSpace(in.Status)
	in.Provider = strings.TrimSpace(in.Provider)
	in.ExternalReference = strings.TrimSpace(in.ExternalReference)
	amount, amountOK := new(big.Rat).SetString(in.Amount)
	if !amountOK || amount.Sign() < 0 || (in.Status != "pending" && in.Status != "paid" && in.Status != "failed" && in.Status != "refunded") || in.Provider == "" {
		return result, subscriptionFailure(apperror.Invalid, "invalid_payment", "Monto, estado y proveedor son obligatorios.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, subscriptionFailure(apperror.Unavailable, "payment_unavailable", "No pudimos registrar el pago.")
	}
	defer tx.Rollback(ctx)
	var subscriptionID, billingCycle, currency string
	var moduleKeys []string
	var currentPeriodEnd *time.Time
	if err = func() error {
		var err error
		subscriptionID, billingCycle, currency, moduleKeys, currentPeriodEnd, err = a.repo.LockPaymentContract(tx.Context(), s, organizationID)
		return err
	}(); err != nil {
		return result, subscriptionFailure(apperror.NotFound, "subscription_not_found", "La empresa todavía no tiene una suscripción.")
	}
	if in.Currency == "" {
		in.Currency = currency
	}
	if in.Currency != currency {
		return result, subscriptionFailure(apperror.Conflict, "payment_currency_mismatch", "El pago debe registrarse en la moneda de la suscripción.")
	}
	var paidAt *time.Time
	if in.PaidAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, in.PaidAt)
		if parseErr != nil {
			return result, subscriptionFailure(apperror.Invalid, "invalid_payment", "La fecha de pago no es válida.")
		}
		paidAt = &parsed
	} else if in.Status == "paid" {
		now := time.Now().UTC()
		paidAt = &now
	}
	now := time.Now().UTC()
	periodStart := now
	if currentPeriodEnd != nil && currentPeriodEnd.After(now) {
		periodStart = *currentPeriodEnd
	}
	periodEnd := domain.SubscriptionPeriodEnd(periodStart, billingCycle)
	var external *string
	if in.ExternalReference != "" {
		external = &in.ExternalReference
	}
	var id string
	err = func() error {
		var err error
		id, err = a.repo.RecordPayment(tx.Context(), s, organizationID, subscriptionID, in, external, periodStart, periodEnd, paidAt)
		return err
	}()
	if err != nil {
		return result, subscriptionFailure(apperror.Conflict, "payment_conflict", "La referencia del pago ya fue registrada.")
	}
	if in.Status == "paid" {
		_, err = a.repo.RenewContract(tx.Context(), s, organizationID, periodStart, periodEnd)
		if err == nil {
			err = a.SyncModules(tx.Context(), organizationID, moduleKeys)
		}
	}
	if err != nil || tx.Commit(ctx) != nil {
		return result, subscriptionFailure(apperror.Unavailable, "payment_unavailable", "No pudimos confirmar el pago.")
	}
	_ = a.audit.Record(ctx, s, "subscription.payment_recorded", "subscription_payment", id)
	return map[string]string{"id": id}, nil

}
func (a *Subscriptions) Create(ctx context.Context, organizationID, administratorID, planID, billingCycle string, termsAccepted bool) error {
	if billingCycle != "monthly" && billingCycle != "annual" {
		return errors.New("invalid billing cycle")
	}
	if !termsAccepted {
		return errors.New("terms not accepted")
	}
	plan, err := a.repo.Plan(ctx, planID, true)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	status := "active"
	price := plan.MonthlyPrice
	var trialStarts, trialEnds *time.Time
	periodEnd := domain.SubscriptionPeriodEnd(now, billingCycle)
	if billingCycle == "annual" {
		price = plan.AnnualPrice
	}
	if plan.TrialDays > 0 {
		status = "trial"
		start := now
		end := now.AddDate(0, 0, plan.TrialDays)
		trialStarts, trialEnds = &start, &end
		periodEnd = end
	}
	err = a.repo.CreateContract(ctx, organizationID, administratorID, plan, billingCycle, price, status, trialStarts, trialEnds, now, periodEnd)
	if err != nil {
		return err
	}
	return a.SyncModules(ctx, organizationID, plan.ModuleKeys)
}
