package postgres

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Usage persists the monthly Concierge counters. Months follow each company's
// timezone; `month` travels as YYYY-MM.
type Usage struct{ pool *pgxpool.Pool }

func NewUsage(pool *pgxpool.Pool) *Usage { return &Usage{pool} }
func (r *Usage) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

var _ application.UsageRepository = (*Usage)(nil)

const currentPeriod = `date_trunc('month',now() AT TIME ZONE o.timezone)::date`

func (r *Usage) Settings(ctx context.Context) (domain.UsageSettings, error) {
	var out domain.UsageSettings
	err := r.db(ctx).QueryRow(ctx, `
		SELECT mode,warn_percent,to_char(updated_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM concierge_usage_settings
		WHERE singleton
	`).Scan(&out.Mode, &out.WarnPercent, &out.UpdatedAt)
	return out, err
}

func (r *Usage) SaveSettings(ctx context.Context, userID string, in domain.UsageSettingsInput) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE concierge_usage_settings
		SET mode=$1,warn_percent=$2,updated_by=NULLIF($3,'')::uuid,updated_at=now()
		WHERE singleton
	`, in.Mode, in.WarnPercent, userID)
	return err
}

func (r *Usage) LockMonth(ctx context.Context, organizationID string) (string, int, error) {
	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO concierge_monthly_usage(organization_id,period_month)
		SELECT o.id,`+currentPeriod+` FROM organizations o WHERE o.id=$1
		ON CONFLICT(organization_id,period_month) DO NOTHING
	`, organizationID); err != nil {
		return "", 0, err
	}
	var month string
	var count int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT to_char(u.period_month,'YYYY-MM'),u.conversations
		FROM concierge_monthly_usage u
		JOIN organizations o ON o.id=u.organization_id
		WHERE u.organization_id=$1 AND u.period_month=`+currentPeriod+`
		FOR UPDATE OF u
	`, organizationID).Scan(&month, &count)
	return month, count, err
}

func (r *Usage) Conversation(ctx context.Context, organizationID, conversationID string) (string, domain.ConversationView, error) {
	var ref string
	var out domain.ConversationView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text,conversation_id,kind,to_char(period_month,'YYYY-MM')
		FROM concierge_conversations
		WHERE organization_id=$1 AND conversation_id=$2
	`, organizationID, conversationID).Scan(&ref, &out.ConversationID, &out.Kind, &out.Month)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", out, application.ErrNotFound
	}
	return ref, out, err
}

func (r *Usage) Limit(ctx context.Context, organizationID string) (domain.Limit, error) {
	var planMax, overrideMax *int
	var hasOverride bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.max_concierge_conversations,x.organization_id IS NOT NULL,x.max_conversations
		FROM organizations o
		LEFT JOIN organization_subscriptions s ON s.organization_id=o.id
		LEFT JOIN subscription_plans p ON p.id=s.plan_id
		LEFT JOIN organization_concierge_limits x ON x.organization_id=o.id
		WHERE o.id=$1
	`, organizationID).Scan(&planMax, &hasOverride, &overrideMax)
	return domain.EffectiveLimit(planMax, hasOverride, overrideMax), err
}

// InsertConversation counts the conversation in the locked month. A concurrent
// insert of the same conversationId counts nothing and returns the existing row.
func (r *Usage) InsertConversation(ctx context.Context, qr application.QRContext, in domain.ConversationInput, month string) (string, error) {
	var ref string
	err := r.db(ctx).QueryRow(ctx, `
		WITH c AS (
			INSERT INTO concierge_conversations(organization_id,location_id,conversation_id,kind,period_month)
			VALUES($1,$2,$3,$4,to_date($5,'YYYY-MM'))
			ON CONFLICT(organization_id,conversation_id) DO NOTHING
			RETURNING id
		), counted AS (
			UPDATE concierge_monthly_usage
			SET conversations=conversations+1,updated_at=now()
			WHERE organization_id=$1 AND period_month=to_date($5,'YYYY-MM') AND EXISTS(SELECT 1 FROM c)
		)
		SELECT id::text FROM c
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, in.ConversationID, in.Kind, month).Scan(&ref)
	if errors.Is(err, pgx.ErrNoRows) {
		ref, _, err = r.Conversation(ctx, qr.Scope.OrganizationID, in.ConversationID)
	}
	return ref, err
}

func (r *Usage) RecordRejected(ctx context.Context, organizationID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO concierge_monthly_usage(organization_id,period_month,rejected_conversations)
		SELECT o.id,`+currentPeriod+`,1 FROM organizations o WHERE o.id=$1
		ON CONFLICT(organization_id,period_month)
		DO UPDATE SET rejected_conversations=concierge_monthly_usage.rejected_conversations+1,updated_at=now()
	`, organizationID)
	return err
}

func (r *Usage) RecordTokens(ctx context.Context, organizationID, conversationRef, month, requestID string, tiers []domain.TierTokens) error {
	names := make([]string, 0, len(tiers))
	inputs := make([]int64, 0, len(tiers))
	outputs := make([]int64, 0, len(tiers))
	for _, t := range tiers {
		names = append(names, t.Tier)
		inputs = append(inputs, t.InputTokens)
		outputs = append(outputs, t.OutputTokens)
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO concierge_token_usage(organization_id,conversation_ref,period_month,request_id,tier,input_tokens,output_tokens)
		SELECT $1,$2,to_date($3,'YYYY-MM'),$4,t.tier,t.input_tokens,t.output_tokens
		FROM unnest($5::text[],$6::bigint[],$7::bigint[]) AS t(tier,input_tokens,output_tokens)
		ON CONFLICT(conversation_ref,request_id,tier) DO NOTHING
	`, organizationID, conversationRef, month, requestID, names, inputs, outputs)
	return err
}

const usageRows = `
	SELECT o.id::text,o.trade_name,to_char(m.period,'YYYY-MM'),p.name,p.max_concierge_conversations,
	       x.organization_id IS NOT NULL,x.max_conversations,
	       COALESCE(u.conversations,0),COALESCE(u.rejected_conversations,0)
	FROM organizations o
	CROSS JOIN LATERAL (SELECT ` + currentPeriod + ` AS period) m
	LEFT JOIN organization_subscriptions s ON s.organization_id=o.id
	LEFT JOIN subscription_plans p ON p.id=s.plan_id
	LEFT JOIN organization_concierge_limits x ON x.organization_id=o.id
	LEFT JOIN concierge_monthly_usage u ON u.organization_id=o.id AND u.period_month=m.period`

func scanUsageRow(row pgx.Row, out *application.UsageRow) error {
	return row.Scan(&out.OrganizationID, &out.OrganizationName, &out.Month, &out.PlanName, &out.PlanMax,
		&out.HasOverride, &out.OverrideMax, &out.Conversations, &out.Rejected)
}

func (r *Usage) UsageRows(ctx context.Context) ([]application.UsageRow, error) {
	rows, err := r.db(ctx).Query(ctx, usageRows+` ORDER BY o.trade_name,o.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.UsageRow{}
	for rows.Next() {
		var item application.UsageRow
		if err = scanUsageRow(rows, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Usage) OrganizationUsage(ctx context.Context, organizationID string) (application.UsageRow, error) {
	var out application.UsageRow
	err := scanUsageRow(r.db(ctx).QueryRow(ctx, usageRows+` WHERE o.id=$1`, organizationID), &out)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	return out, err
}

// Tokens always lists both tiers, lite first.
func (r *Usage) Tokens(ctx context.Context, organizationID, month string) ([]domain.TierTokens, error) {
	out := []domain.TierTokens{{Tier: domain.TierLite}, {Tier: domain.TierFlash}}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT tier,COALESCE(sum(input_tokens),0)::bigint,COALESCE(sum(output_tokens),0)::bigint
		FROM concierge_token_usage
		WHERE organization_id=$1 AND period_month=to_date($2,'YYYY-MM')
		GROUP BY tier
	`, organizationID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.TierTokens
		if err = rows.Scan(&item.Tier, &item.InputTokens, &item.OutputTokens); err != nil {
			return nil, err
		}
		for i := range out {
			if out[i].Tier == item.Tier {
				out[i] = item
			}
		}
	}
	return out, rows.Err()
}

// History lists the months with activity among the last `months`, newest first.
func (r *Usage) History(ctx context.Context, organizationID string, months int) ([]domain.UsageMonth, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT to_char(u.period_month,'YYYY-MM'),u.conversations,u.rejected_conversations,
		       COALESCE(sum(t.input_tokens),0)::bigint,COALESCE(sum(t.output_tokens),0)::bigint
		FROM organizations o
		JOIN concierge_monthly_usage u ON u.organization_id=o.id
		LEFT JOIN concierge_token_usage t ON t.organization_id=u.organization_id AND t.period_month=u.period_month
		WHERE o.id=$1
		  AND u.period_month>(`+currentPeriod+`-make_interval(months=>$2))::date
		  AND u.period_month<=`+currentPeriod+`
		GROUP BY u.period_month,u.conversations,u.rejected_conversations
		ORDER BY u.period_month DESC
	`, organizationID, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.UsageMonth{}
	for rows.Next() {
		var item domain.UsageMonth
		if err = rows.Scan(&item.Month, &item.Conversations, &item.RejectedConversations, &item.InputTokens, &item.OutputTokens); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Usage) SaveOverride(ctx context.Context, userID, organizationID string, unlimited bool, max *int) error {
	if unlimited {
		max = nil
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO organization_concierge_limits(organization_id,max_conversations,updated_by)
		VALUES($1,$2,NULLIF($3,'')::uuid)
		ON CONFLICT(organization_id)
		DO UPDATE SET max_conversations=EXCLUDED.max_conversations,updated_by=EXCLUDED.updated_by,updated_at=now()
	`, organizationID, max, userID)
	return err
}

func (r *Usage) DeleteOverride(ctx context.Context, organizationID string) error {
	_, err := r.db(ctx).Exec(ctx, `DELETE FROM organization_concierge_limits WHERE organization_id=$1`, organizationID)
	return err
}
