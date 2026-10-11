package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope
type Platform struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Platform { return &Platform{pool} }
func (r *Platform) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}
func platformError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "23505" {
		return application.ErrConflict
	}
	return err
}

var _ application.Repository = (*Platform)(nil)

const platformOrganizationsBase = `
	WITH base AS (
		SELECT o.id,o.legal_name,o.trade_name,COALESCE(o.tax_id,'') AS tax_id,o.active,o.created_at,
		       s.id AS subscription_id,p.id AS plan_id,p.code AS plan_code,p.name AS plan_name,
		       s.billing_cycle,s.price_amount,s.currency,s.status,s.renews_at,s.trial_ends_at,s.auto_renew,
		       lp.amount AS last_amount,lp.currency AS last_currency,lp.paid_at AS last_paid_at,
		       CASE
		         WHEN s.id IS NULL THEN 'none'
		         WHEN s.status='cancelled' THEN 'cancelled'
		         WHEN s.status='past_due' THEN 'overdue'
		         WHEN s.status='trial' THEN 'trial'
		         WHEN s.renews_at IS NOT NULL AND s.renews_at < now()+make_interval(days=>$1) THEN 'due_soon'
		         ELSE 'up_to_date'
		       END AS standing
		FROM organizations o
		LEFT JOIN organization_subscriptions s ON s.organization_id=o.id
		LEFT JOIN subscription_plans p ON p.id=s.plan_id
		LEFT JOIN LATERAL (
			SELECT sp.amount,sp.currency,sp.paid_at
			FROM subscription_payments sp
			WHERE sp.organization_id=o.id AND sp.status='paid'
			ORDER BY sp.paid_at DESC NULLS LAST,sp.created_at DESC
			LIMIT 1
		) lp ON true
	)`
const utcTimestampFormat = `'YYYY-MM-DD"T"HH24:MI:SS"Z"'`

func (r *Platform) MarkPastDue(ctx context.Context, s scope) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE organization_subscriptions
		SET status='past_due',updated_at=now()
		WHERE status IN ('trial','active') AND renews_at IS NOT NULL AND renews_at<now()
	`)
	return tag.RowsAffected(), platformError(err)
}
func (r *Platform) Organizations(ctx context.Context, f application.Filter) (application.OrganizationPage, error) {
	var out application.OrganizationPage
	page, size, q, planID, standing := f.Page, f.Size, f.Query, f.PlanID, f.Standing
	summary := map[string]int{"total": 0}
	for _, option := range domain.PaymentStandingOptions {
		summary[option.Value] = 0
	}
	summaryRows, err := r.db(ctx).Query(ctx, platformOrganizationsBase+` SELECT standing,count(*) FROM base GROUP BY standing`, domain.SubscriptionDueSoonDays)
	if err != nil {
		return out, err
	}
	for summaryRows.Next() {
		var key string
		var count int
		if err = summaryRows.Scan(&key, &count); err != nil {
			summaryRows.Close()
			return out, err
		}
		summary[key] = count
		summary["total"] += count
	}
	summaryRows.Close()

	rows, err := r.db(ctx).Query(ctx, platformOrganizationsBase+`
		SELECT id,legal_name,trade_name,tax_id,active,
		       to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       subscription_id IS NOT NULL,COALESCE(plan_id::text,''),COALESCE(plan_code,''),COALESCE(plan_name,''),
		       COALESCE(billing_cycle,''),COALESCE(price_amount::text,''),COALESCE(currency,''),COALESCE(status,''),
		       CASE WHEN renews_at IS NULL THEN NULL ELSE to_char(renews_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       CASE WHEN trial_ends_at IS NULL THEN NULL ELSE to_char(trial_ends_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       COALESCE(auto_renew,false),
		       last_amount IS NOT NULL,COALESCE(last_amount::text,''),COALESCE(last_currency,''),
		       CASE WHEN last_paid_at IS NULL THEN NULL ELSE to_char(last_paid_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') END,
		       standing,
		       count(*) OVER()
		FROM base
		WHERE ($2='' OR trade_name ILIKE '%'||$2||'%' OR legal_name ILIKE '%'||$2||'%' OR tax_id ILIKE '%'||$2||'%')
		  AND ($3='' OR plan_id::text=$3)
		  AND ($4='' OR standing=$4)
		ORDER BY CASE standing WHEN 'overdue' THEN 0 WHEN 'due_soon' THEN 1 WHEN 'trial' THEN 2 WHEN 'up_to_date' THEN 3 WHEN 'none' THEN 4 ELSE 5 END,
		         renews_at NULLS LAST,trade_name
		LIMIT $5 OFFSET $6
	`, domain.SubscriptionDueSoonDays, q, planID, standing, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.PlatformOrganizationView{}
	total := 0
	for rows.Next() {
		var item domain.PlatformOrganizationView
		var sub domain.PlatformOrganizationSubscription
		var payment domain.PlatformOrganizationPayment
		var hasSubscription, hasPayment bool
		if err = rows.Scan(
			&item.ID, &item.LegalName, &item.TradeName, &item.TaxID, &item.Active, &item.CreatedAt,
			&hasSubscription, &sub.Plan.ID, &sub.Plan.Code, &sub.Plan.Name,
			&sub.BillingCycle, &sub.PriceAmount, &sub.Currency, &sub.Status,
			&sub.RenewsAt, &sub.TrialEndsAt, &sub.AutoRenew,
			&hasPayment, &payment.Amount, &payment.Currency, &payment.PaidAt,
			&item.PaymentStanding, &total,
		); err != nil {
			return out, err
		}
		if hasSubscription {
			item.Subscription = &sub
		}
		if hasPayment {
			item.LastPayment = &payment
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}

	planOptions := []domain.OptionView{}
	planRows, err := r.db(ctx).Query(ctx, `SELECT id::text,name FROM subscription_plans ORDER BY active DESC,monthly_price,name`)
	if err == nil {
		defer planRows.Close()
		for planRows.Next() {
			var option domain.OptionView
			if planRows.Scan(&option.Value, &option.Label) == nil {
				planOptions = append(planOptions, option)
			}
		}
	}

	out.Items = items
	out.Total = total
	out.Page = page
	out.PageSize = size
	out.Summary = summary
	out.PlanOptions = planOptions
	return out, nil
}
func (r *Platform) Organization(ctx context.Context, id string) (domain.PlatformOrganizationDetail, error) {
	var out domain.PlatformOrganizationDetail
	err := r.db(ctx).QueryRow(ctx, `
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
		return out, application.ErrNotFound
	}
	if err != nil {
		return out, err
	}

	if err = r.db(ctx).QueryRow(ctx, platformOrganizationsBase+` SELECT standing FROM base WHERE id=$2`, domain.SubscriptionDueSoonDays, id).Scan(&out.PaymentStanding); err != nil {
		return out, err
	}

	var sub domain.PlatformOrganizationDetailSubscription
	err = r.db(ctx).QueryRow(ctx, `
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
		return out, err
	}

	if err = r.db(ctx).QueryRow(ctx, `
		SELECT count(*),COALESCE(sum(amount),0)::numeric(12,2)::text,
		       CASE WHEN max(paid_at) IS NULL THEN NULL ELSE to_char(max(paid_at) AT TIME ZONE 'UTC',`+utcTimestampFormat+`) END
		FROM subscription_payments
		WHERE organization_id=$1 AND status='paid'
	`, id).Scan(&out.Payments.PaidCount, &out.Payments.TotalPaid, &out.Payments.LastPaidAt); err != nil {
		return out, err
	}

	out.Administrators = []domain.PlatformOrganizationAdmin{}
	adminRows, err := r.db(ctx).Query(ctx, `
		SELECT DISTINCT u.full_name,u.email,u.active
		FROM users u
		JOIN user_roles ur ON ur.user_id=u.id
		JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=u.organization_id
		WHERE u.organization_id=$1 AND NOT u.platform_admin AND ro.system_key='administrator'
		ORDER BY u.active DESC,u.full_name
	`, id)
	if err != nil {
		return out, err
	}
	for adminRows.Next() {
		var item domain.PlatformOrganizationAdmin
		if err = adminRows.Scan(&item.Name, &item.Email, &item.Active); err != nil {
			adminRows.Close()
			return out, err
		}
		out.Administrators = append(out.Administrators, item)
	}
	adminRows.Close()

	out.Locations = []domain.PlatformOrganizationLocation{}
	locationRows, err := r.db(ctx).Query(ctx, `
		SELECT id,name,code,address,phone,active
		FROM locations
		WHERE organization_id=$1
		ORDER BY active DESC,created_at
	`, id)
	if err != nil {
		return out, err
	}
	defer locationRows.Close()
	for locationRows.Next() {
		var item domain.PlatformOrganizationLocation
		if err = locationRows.Scan(&item.ID, &item.Name, &item.Code, &item.Address, &item.Phone, &item.Active); err != nil {
			return out, err
		}
		out.Locations = append(out.Locations, item)
	}
	return out, locationRows.Err()
}
func (r *Platform) Channels(ctx context.Context) ([]domain.PlatformWhatsAppChannel, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT w.id,c.code,c.name,w.phone_number,COALESCE(w.display_name,''),w.active
		FROM platform_whatsapp_channels w
		JOIN platform_countries c ON c.code=w.country_code
		ORDER BY c.name,w.created_at,w.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PlatformWhatsAppChannel{}
	for rows.Next() {
		var item domain.PlatformWhatsAppChannel
		if err := rows.Scan(&item.ID, &item.CountryCode, &item.CountryName, &item.PhoneNumber, &item.DisplayName, &item.Active); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, rows.Err()
}
func (r *Platform) CreateChannel(ctx context.Context, s scope, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error) {
	var item domain.PlatformWhatsAppChannel
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO platform_whatsapp_channels(country_code,phone_number,display_name,active)
		VALUES($1,$2,$3,$4)
		RETURNING id,country_code,(SELECT name FROM platform_countries WHERE code=$1),phone_number,COALESCE(display_name,''),active`,
		in.CountryCode, in.PhoneNumber, in.DisplayName, in.Active).
		Scan(&item.ID, &item.CountryCode, &item.CountryName, &item.PhoneNumber, &item.DisplayName, &item.Active)
	return item, platformError(err)
}
func (r *Platform) UpdateChannel(ctx context.Context, s scope, id string, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error) {
	var item domain.PlatformWhatsAppChannel
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE platform_whatsapp_channels w SET
		  country_code=$2,phone_number=$3,display_name=$4,active=$5,updated_at=now()
		WHERE w.id=$1
		RETURNING w.id,w.country_code,(SELECT name FROM platform_countries WHERE code=w.country_code),w.phone_number,COALESCE(w.display_name,''),w.active`,
		id, in.CountryCode, in.PhoneNumber, in.DisplayName, in.Active).
		Scan(&item.ID, &item.CountryCode, &item.CountryName, &item.PhoneNumber, &item.DisplayName, &item.Active)
	return item, platformError(err)
}
func (r *Platform) CountryExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM platform_countries WHERE code=$1 AND active)`, code).Scan(&exists)
	return exists, err
}
func (r *Platform) PlatformAdmin(ctx context.Context, id string) (bool, error) {
	var allowed bool
	err := r.db(ctx).QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, id).Scan(&allowed)
	return allowed, err
}
