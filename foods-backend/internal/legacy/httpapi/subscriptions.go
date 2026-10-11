package httpapi

import (
	"context"
	"errors"
	subscriptiondomain "github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var errSubscriptionLimit = errors.New("subscription limit reached")

type subscriptionPlanView = subscriptiondomain.SubscriptionPlanView

type subscriptionPlanInput = subscriptiondomain.SubscriptionPlanInput

type subscriptionPaymentView = subscriptiondomain.SubscriptionPaymentView

type subscriptionOrganizationView = subscriptiondomain.SubscriptionOrganizationView

type organizationSubscriptionView = subscriptiondomain.OrganizationSubscriptionView

type subscriptionUsage = subscriptiondomain.SubscriptionUsage

type subscriptionUpdateInput = subscriptiondomain.SubscriptionUpdateInput

type subscriptionPaymentInput = subscriptiondomain.SubscriptionPaymentInput

var planCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,39}$`)

func normalizePlanInput(in *subscriptionPlanInput) bool {
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
	if _, ok := currencyByCode(in.Currency); !ok {
		return false
	}
	monthly, monthlyOK := new(big.Rat).SetString(in.MonthlyPrice)
	annual, annualOK := new(big.Rat).SetString(in.AnnualPrice)
	if !monthlyOK || !annualOK || monthly.Sign() < 0 || annual.Sign() < 0 || in.TrialDays < 0 || in.TrialDays > 365 {
		return false
	}
	if (in.MaxLocations != nil && *in.MaxLocations < 1) || (in.MaxUsers != nil && *in.MaxUsers < 1) {
		return false
	}
	seen := map[string]bool{}
	modules := make([]string, 0, len(in.ModuleKeys))
	for _, raw := range in.ModuleKeys {
		key := strings.TrimSpace(raw)
		if key == "" || seen[key] || !moduleCanActivate(key) {
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

func scanSubscriptionPlan(row pgx.Row, out *subscriptionPlanView) error {
	return row.Scan(
		&out.ID, &out.Code, &out.Name, &out.Description, &out.Currency,
		&out.MonthlyPrice, &out.AnnualPrice, &out.TrialDays,
		&out.MaxLocations, &out.MaxUsers, &out.ModuleKeys, &out.TermsVersion, &out.Active,
	)
}

func readSubscriptionPlan(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string, activeOnly bool) (subscriptionPlanView, error) {
	var out subscriptionPlanView
	err := scanSubscriptionPlan(q.QueryRow(ctx, `
		SELECT id,code,name,description,currency,monthly_price::text,annual_price::text,
		       trial_days,max_locations,max_users,module_keys,terms_version,active
		FROM subscription_plans
		WHERE id=$1 AND (NOT $2 OR active)
	`, id, activeOnly), &out)
	return out, err
}

func (a *API) listSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.ListSubscriptionPlans(w, r)
}

func (a *API) listAvailableSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.ListAvailableSubscriptionPlans(w, r)
}

func (a *API) createSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.CreateSubscriptionPlan(w, r)
}

func (a *API) updateSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.UpdateSubscriptionPlan(w, r)
}

func subscriptionPeriodEnd(start time.Time, cycle string) time.Time {
	if cycle == "annual" {
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 1, 0)
}

func createOrganizationSubscription(ctx context.Context, tx pgx.Tx, organizationID, administratorID, planID, billingCycle string, termsAccepted bool) error {
	if billingCycle != "monthly" && billingCycle != "annual" {
		return errors.New("invalid billing cycle")
	}
	if !termsAccepted {
		return errors.New("terms not accepted")
	}
	plan, err := readSubscriptionPlan(ctx, tx, planID, true)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	status := "active"
	price := plan.MonthlyPrice
	var trialStarts, trialEnds *time.Time
	periodEnd := subscriptionPeriodEnd(now, billingCycle)
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
	_, err = tx.Exec(ctx, `
		INSERT INTO organization_subscriptions(
		  organization_id,plan_id,billing_cycle,price_amount,currency,status,
		  trial_starts_at,trial_ends_at,current_period_starts_at,current_period_ends_at,renews_at,
		  auto_renew,terms_version,terms_accepted_at,terms_accepted_by
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,true,$11,$9,$12)
	`, organizationID, plan.ID, billingCycle, price, plan.Currency, status, trialStarts, trialEnds, now, periodEnd, plan.TermsVersion, administratorID)
	if err != nil {
		return err
	}
	return syncOrganizationModulesForPlan(ctx, tx, organizationID, plan.ModuleKeys)
}

var organizationIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// subscriptionOrganizationID resuelve la empresa de la suscripción: la indicada en la ruta de
// Plataforma (/v1/platform/organizations/{id}/...) o, en su ausencia, la de la sesión.

func (a *API) getOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.GetOrganizationSubscription(w, r)
}

func ensureSubscriptionCapacity(ctx context.Context, tx pgx.Tx, organizationID, resource string) error {
	var max *int
	switch resource {
	case "locations":
		if err := tx.QueryRow(ctx, `
			SELECT p.max_locations
			FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id
			WHERE s.organization_id=$1
			FOR UPDATE OF s
		`, organizationID).Scan(&max); err != nil {
			return err
		}
		if max == nil {
			return nil
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`, organizationID).Scan(&count); err != nil {
			return err
		}
		if count >= *max {
			return errSubscriptionLimit
		}
	case "users":
		if err := tx.QueryRow(ctx, `
			SELECT p.max_users
			FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id
			WHERE s.organization_id=$1
			FOR UPDATE OF s
		`, organizationID).Scan(&max); err != nil {
			return err
		}
		if max == nil {
			return nil
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, organizationID).Scan(&count); err != nil {
			return err
		}
		if count >= *max {
			return errSubscriptionLimit
		}
	default:
		return errors.New("unknown subscription resource")
	}
	return nil
}

func (a *API) updateOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.UpdateOrganizationSubscription(w, r)
}
func (a *API) recordSubscriptionPayment(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.RecordSubscriptionPayment(w, r)
}
