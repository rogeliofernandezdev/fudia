package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type scope = requestscope.Scope
type Subscriptions struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Subscriptions { return &Subscriptions{pool} }
func (r *Subscriptions) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}
func subscriptionError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.SubscriptionRepository = (*Subscriptions)(nil)

func scanSubscriptionPlan(row pgx.Row, out *domain.SubscriptionPlanView) error {
	return row.Scan(
		&out.ID, &out.Code, &out.Name, &out.Description, &out.Currency,
		&out.MonthlyPrice, &out.AnnualPrice, &out.TrialDays,
		&out.MaxLocations, &out.MaxUsers, &out.ModuleKeys, &out.TermsVersion, &out.Active,
	)
}
func (r *Subscriptions) Plan(ctx context.Context, id string, activeOnly bool) (domain.SubscriptionPlanView, error) {
	var out domain.SubscriptionPlanView
	err := scanSubscriptionPlan(r.db(ctx).QueryRow(ctx, `
		SELECT id,code,name,description,currency,monthly_price::text,annual_price::text,
		       trial_days,max_locations,max_users,module_keys,terms_version,active
		FROM subscription_plans
		WHERE id=$1 AND (NOT $2 OR active)
	`, id, activeOnly), &out)
	return out, subscriptionError(err)
}
func (r *Subscriptions) PlanOrganizations(ctx context.Context, id string) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT organization_id FROM organization_subscriptions WHERE plan_id=$1 AND status<>'cancelled'`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func (r *Subscriptions) SaveModule(ctx context.Context, org, key string, active bool) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO organization_modules(organization_id,module_key,active) VALUES($1,$2,$3) ON CONFLICT(organization_id,module_key) DO UPDATE SET active=EXCLUDED.active,updated_at=now()`, org, key, active)
	return err
}

func (r *Subscriptions) CreatePlan(ctx context.Context, s scope, in domain.SubscriptionPlanInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO subscription_plans(code,name,description,currency,monthly_price,annual_price,trial_days,max_locations,max_users,module_keys,terms_version,active)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING id
		`, in.Code, in.Name, in.Description, in.Currency, in.MonthlyPrice, in.AnnualPrice, in.TrialDays, in.MaxLocations, in.MaxUsers, in.ModuleKeys, in.TermsVersion, in.Active).Scan(&id)
	return id, subscriptionError(err)
}
func (r *Subscriptions) PlanLimitsExceeded(ctx context.Context, s scope, id string, in domain.SubscriptionPlanInput) (bool, error) {
	var exceeds bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
			  SELECT 1
			  FROM organization_subscriptions s
			  WHERE s.plan_id=$1
			    AND (
			      ($2::integer IS NOT NULL AND (SELECT count(*) FROM locations l WHERE l.organization_id=s.organization_id AND l.active)>$2)
			      OR
			      ($3::integer IS NOT NULL AND (SELECT count(*) FROM users u WHERE u.organization_id=s.organization_id AND u.active AND NOT u.platform_admin)>$3)
			    )
			)
		`, id, in.MaxLocations, in.MaxUsers).Scan(&exceeds)
	return exceeds, subscriptionError(err)
}
func (r *Subscriptions) UpdatePlan(ctx context.Context, s scope, id string, in domain.SubscriptionPlanInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
				UPDATE subscription_plans
				SET code=$2,name=$3,description=$4,currency=$5,monthly_price=$6,annual_price=$7,
				    trial_days=$8,max_locations=$9,max_users=$10,module_keys=$11,terms_version=$12,active=$13,updated_at=now()
				WHERE id=$1 AND code<>'legacy'
			`, id, in.Code, in.Name, in.Description, in.Currency, in.MonthlyPrice, in.AnnualPrice, in.TrialDays, in.MaxLocations, in.MaxUsers, in.ModuleKeys, in.TermsVersion, in.Active)
	return tag.RowsAffected(), subscriptionError(err)
}
func (r *Subscriptions) Plans(ctx context.Context) ([]domain.SubscriptionPlanView, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT id,code,name,description,currency,monthly_price::text,annual_price::text,
		       trial_days,max_locations,max_users,module_keys,terms_version,active
		FROM subscription_plans
		ORDER BY active DESC,
		         CASE code WHEN 'emprende' THEN 1 WHEN 'impulso' THEN 2 WHEN 'escala' THEN 3 ELSE 4 END,
		         name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SubscriptionPlanView{}
	for rows.Next() {
		var item domain.SubscriptionPlanView
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Currency, &item.MonthlyPrice, &item.AnnualPrice, &item.TrialDays, &item.MaxLocations, &item.MaxUsers, &item.ModuleKeys, &item.TermsVersion, &item.Active); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Subscriptions) AvailablePlans(ctx context.Context) ([]domain.SubscriptionPlanView, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT id,code,name,description,currency,monthly_price::text,annual_price::text,
		       trial_days,max_locations,max_users,module_keys,terms_version,active
		FROM subscription_plans
		WHERE active AND code<>'legacy'
		ORDER BY CASE code WHEN 'emprende' THEN 1 WHEN 'impulso' THEN 2 WHEN 'escala' THEN 3 ELSE 4 END,name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SubscriptionPlanView{}
	for rows.Next() {
		var item domain.SubscriptionPlanView
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Currency, &item.MonthlyPrice, &item.AnnualPrice, &item.TrialDays, &item.MaxLocations, &item.MaxUsers, &item.ModuleKeys, &item.TermsVersion, &item.Active); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Subscriptions) OrganizationExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)`, id).Scan(&exists)
	return exists, err
}
func (r *Subscriptions) MarkPastDue(ctx context.Context, s scope, organizationID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE organization_subscriptions
		SET status='past_due',updated_at=now()
		WHERE organization_id=$1 AND status IN ('trial','active') AND renews_at IS NOT NULL AND renews_at<now()
	`, organizationID)
	return tag.RowsAffected(), subscriptionError(err)
}
func (r *Subscriptions) Subscription(ctx context.Context, organizationID string) (domain.OrganizationSubscriptionView, error) {
	var out domain.OrganizationSubscriptionView
	var plan domain.SubscriptionPlanView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT s.id,
		       p.id,p.code,p.name,p.description,p.currency,p.monthly_price::text,p.annual_price::text,p.trial_days,p.max_locations,p.max_users,p.module_keys,p.terms_version,p.active,
		       s.billing_cycle,s.price_amount::text,s.currency,s.status,
		       CASE WHEN s.trial_starts_at IS NULL THEN NULL ELSE to_char(s.trial_starts_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.trial_ends_at IS NULL THEN NULL ELSE to_char(s.trial_ends_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.current_period_starts_at IS NULL THEN NULL ELSE to_char(s.current_period_starts_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.current_period_ends_at IS NULL THEN NULL ELSE to_char(s.current_period_ends_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN s.renews_at IS NULL THEN NULL ELSE to_char(s.renews_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       s.auto_renew,s.terms_version,
		       CASE WHEN s.terms_accepted_at IS NULL THEN NULL ELSE to_char(s.terms_accepted_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END
		FROM organization_subscriptions s
		JOIN subscription_plans p ON p.id=s.plan_id
		WHERE s.organization_id=$1
	`, organizationID).Scan(
		&out.ID,
		&plan.ID, &plan.Code, &plan.Name, &plan.Description, &plan.Currency, &plan.MonthlyPrice, &plan.AnnualPrice, &plan.TrialDays, &plan.MaxLocations, &plan.MaxUsers, &plan.ModuleKeys, &plan.TermsVersion, &plan.Active,
		&out.BillingCycle, &out.PriceAmount, &out.Currency, &out.Status,
		&out.TrialStartsAt, &out.TrialEndsAt, &out.CurrentPeriodStartsAt, &out.CurrentPeriodEndsAt, &out.RenewsAt,
		&out.AutoRenew, &out.TermsVersion, &out.TermsAcceptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, subscriptionError(err)
	}
	if err != nil {
		return out, subscriptionError(err)
	}
	out.Plan = plan
	out.Organization.ID = organizationID
	_ = r.db(ctx).QueryRow(ctx, `SELECT trade_name,legal_name,COALESCE(tax_id,''),active FROM organizations WHERE id=$1`, organizationID).Scan(&out.Organization.TradeName, &out.Organization.LegalName, &out.Organization.TaxID, &out.Organization.Active)
	_ = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`, organizationID).Scan(&out.Usage.Locations)
	_ = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, organizationID).Scan(&out.Usage.Users)

	out.Payments = []domain.SubscriptionPaymentView{}
	rows, paymentErr := r.db(ctx).Query(ctx, `
		SELECT id,amount::text,currency,status,provider,external_reference,
		       CASE WHEN paid_at IS NULL THEN NULL ELSE to_char(paid_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM subscription_payments
		WHERE organization_id=$1
		ORDER BY created_at DESC
		LIMIT 10
	`, organizationID)
	if paymentErr == nil {
		defer rows.Close()
		for rows.Next() {
			var item domain.SubscriptionPaymentView
			if rows.Scan(&item.ID, &item.Amount, &item.Currency, &item.Status, &item.Provider, &item.ExternalReference, &item.PaidAt, &item.CreatedAt) == nil {
				out.Payments = append(out.Payments, item)
			}
		}
	}
	return out, nil
}
func (r *Subscriptions) LockContract(ctx context.Context, s scope, organizationID string) (string, string, string, string, *string, *time.Time, *time.Time, *time.Time, *time.Time, *time.Time, *time.Time, error) {
	var currentPlanID string
	var currentCycle string
	var currentStatus string
	var currentPrice string
	var currentTerms *string
	var currentTrialStarts *time.Time
	var currentTrialEnds *time.Time
	var currentPeriodStarts *time.Time
	var currentPeriodEnds *time.Time
	var currentRenews *time.Time
	var currentCancelledAt *time.Time
	err := r.db(ctx).QueryRow(ctx, `
		SELECT plan_id,billing_cycle,status,price_amount::text,terms_version,
		       trial_starts_at,trial_ends_at,current_period_starts_at,current_period_ends_at,renews_at,cancelled_at
		FROM organization_subscriptions
		WHERE organization_id=$1
		FOR UPDATE
	`, organizationID).Scan(&currentPlanID, &currentCycle, &currentStatus, &currentPrice, &currentTerms, &currentTrialStarts, &currentTrialEnds, &currentPeriodStarts, &currentPeriodEnds, &currentRenews, &currentCancelledAt)
	return currentPlanID, currentCycle, currentStatus, currentPrice, currentTerms, currentTrialStarts, currentTrialEnds, currentPeriodStarts, currentPeriodEnds, currentRenews, currentCancelledAt, subscriptionError(err)
}
func (r *Subscriptions) ActiveLocations(ctx context.Context, s scope, organizationID string) (int, error) {
	var locationCount int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM locations WHERE organization_id=$1 AND active`, organizationID).Scan(&locationCount)
	return locationCount, subscriptionError(err)
}
func (r *Subscriptions) ActiveUsers(ctx context.Context, s scope, organizationID string) (int, error) {
	var userCount int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM users WHERE organization_id=$1 AND active AND NOT platform_admin`, organizationID).Scan(&userCount)
	return userCount, subscriptionError(err)
}
func (r *Subscriptions) UpdateContract(ctx context.Context, s scope, organizationID string, plan domain.SubscriptionPlanView, in domain.SubscriptionUpdateInput, price string, trialStarts *time.Time, trialEnds *time.Time, periodStarts *time.Time, periodEnds *time.Time, renews *time.Time, termsVersion *string, termsAcceptedAt *time.Time, termsAcceptedBy *string, cancelledAt *time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE organization_subscriptions
		SET plan_id=$2,billing_cycle=$3,price_amount=$4,currency=$5,status=$6,
		    trial_starts_at=$7,trial_ends_at=$8,current_period_starts_at=$9,current_period_ends_at=$10,renews_at=$11,
		    auto_renew=$12,terms_version=COALESCE($13,terms_version),
		    terms_accepted_at=COALESCE($14,terms_accepted_at),terms_accepted_by=COALESCE($15,terms_accepted_by),
		    cancelled_at=$16,updated_at=now()
		WHERE organization_id=$1
	`, organizationID, plan.ID, in.BillingCycle, price, plan.Currency, in.Status, trialStarts, trialEnds, periodStarts, periodEnds, renews, in.AutoRenew, termsVersion, termsAcceptedAt, termsAcceptedBy, cancelledAt)
	return tag.RowsAffected(), subscriptionError(err)
}
func (r *Subscriptions) LockPaymentContract(ctx context.Context, s scope, organizationID string) (string, string, string, []string, *time.Time, error) {
	var subscriptionID string
	var billingCycle string
	var currency string
	var moduleKeys []string
	var currentPeriodEnd *time.Time
	err := r.db(ctx).QueryRow(ctx, `SELECT s.id,s.billing_cycle,s.currency,p.module_keys,s.current_period_ends_at FROM organization_subscriptions s JOIN subscription_plans p ON p.id=s.plan_id WHERE s.organization_id=$1 FOR UPDATE OF s`, organizationID).Scan(&subscriptionID, &billingCycle, &currency, &moduleKeys, &currentPeriodEnd)
	return subscriptionID, billingCycle, currency, moduleKeys, currentPeriodEnd, subscriptionError(err)
}
func (r *Subscriptions) RecordPayment(ctx context.Context, s scope, organizationID string, subscriptionID string, in domain.SubscriptionPaymentInput, external *string, periodStart time.Time, periodEnd time.Time, paidAt *time.Time) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO subscription_payments(organization_id,subscription_id,amount,currency,status,provider,external_reference,period_starts_at,period_ends_at,paid_at,recorded_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id
	`, organizationID, subscriptionID, in.Amount, in.Currency, in.Status, in.Provider, external, periodStart, periodEnd, paidAt, s.UserID).Scan(&id)
	return id, subscriptionError(err)
}
func (r *Subscriptions) RenewContract(ctx context.Context, s scope, organizationID string, periodStart time.Time, periodEnd time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			UPDATE organization_subscriptions
			SET status='active',current_period_starts_at=$2,current_period_ends_at=$3,renews_at=$3,cancelled_at=NULL,updated_at=now()
			WHERE organization_id=$1
		`, organizationID, periodStart, periodEnd)
	return tag.RowsAffected(), subscriptionError(err)
}
func (r *Subscriptions) CreateContract(ctx context.Context, organizationID string, administratorID string, plan domain.SubscriptionPlanView, billingCycle string, price string, status string, trialStarts *time.Time, trialEnds *time.Time, now time.Time, periodEnd time.Time) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO organization_subscriptions(
		  organization_id,plan_id,billing_cycle,price_amount,currency,status,
		  trial_starts_at,trial_ends_at,current_period_starts_at,current_period_ends_at,renews_at,
		  auto_renew,terms_version,terms_accepted_at,terms_accepted_by
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,true,$11,$9,$12)
	`, organizationID, plan.ID, billingCycle, price, plan.Currency, status, trialStarts, trialEnds, now, periodEnd, plan.TermsVersion, administratorID)
	return err
}
