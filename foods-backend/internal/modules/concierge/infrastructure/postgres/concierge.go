package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type scope = requestscope.Scope
type Concierge struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Concierge { return &Concierge{pool} }
func (r *Concierge) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}
func conciergeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.Repository = (*Concierge)(nil)

func (r *Concierge) ResolveQR(ctx context.Context, token string) (application.QRContext, error) {
	var out application.QRContext
	err := r.db(ctx).QueryRow(ctx, `
		SELECT t.organization_id::text,t.location_id::text,t.id::text,t.name,
		       o.trade_name,l.name,p.currency_symbol
		FROM tables t
		JOIN organizations o ON o.id=t.organization_id AND o.active
		JOIN locations l ON l.id=t.location_id AND l.organization_id=t.organization_id AND l.active
		JOIN organization_modules om
		  ON om.organization_id=t.organization_id
		 AND om.module_key='whatsapp_bot'
		 AND om.active
		JOIN organization_fiscal_profiles p
		  ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id AND p.active
		WHERE t.qr_token=$1 AND t.active AND t.qr_enabled
		LIMIT 1
	`, strings.TrimSpace(token)).Scan(
		&out.Scope.OrganizationID,
		&out.Scope.LocationID,
		&out.TableID,
		&out.TableName,
		&out.OrganizationName,
		&out.LocationName,
		&out.CurrencySymbol,
	)
	if err != nil {
		return application.QRContext{}, conciergeError(err)
	}
	out.Scope.Name = "Fudia Concierge"
	return out, nil
}
func (r *Concierge) RecordBillRequest(ctx context.Context, s scope, qr application.QRContext, out orderdomain.Order, in domain.BillRequestInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO audit_log(
		  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
		)
		VALUES($1,$2,NULL,'concierge.bill.requested','order',$3,
		  jsonb_build_object(
		    'source','fudia_concierge',
		    'conversationId',$4::text,
		    'tableId',$5::text,
		    'customerPhonePresent',($6::text<>'')
		  ))
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, out.ID, in.ConversationID, qr.TableID, in.CustomerPhone)
	return tag.RowsAffected(), conciergeError(err)
}
func (r *Concierge) Settings(ctx context.Context, s scope) (domain.SettingsView, error) {
	var out domain.SettingsView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.trade_name,l.name,COALESCE(om.active,false)
		FROM organizations o
		JOIN locations l
		  ON l.organization_id=o.id
		 AND l.id=$2
		 AND l.active
		LEFT JOIN organization_modules om
		  ON om.organization_id=o.id
		 AND om.module_key='whatsapp_bot'
		WHERE o.id=$1 AND o.active
	`, s.OrganizationID, s.LocationID).Scan(
		&out.OrganizationName,
		&out.LocationName,
		&out.Active,
	)
	return out, conciergeError(err)
}

const conciergeHandoffColumns = `
	h.id::text,h.table_id::text,t.name,h.conversation_id,h.customer_phone,h.reason,h.status,
	to_char(h.requested_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
	CASE WHEN h.resolved_at IS NULL THEN NULL ELSE to_char(h.resolved_at,'YYYY-MM-DD"T"HH24:MI:SSOF') END,
	u.full_name
`

func scanConciergeHandoff(row pgx.Row) (domain.HandoffView, error) {
	var out domain.HandoffView
	err := row.Scan(
		&out.ID, &out.TableID, &out.TableName, &out.ConversationID, &out.CustomerPhone,
		&out.Reason, &out.Status, &out.RequestedAt, &out.ResolvedAt, &out.ResolvedByName,
	)
	return out, err
}
func (r *Concierge) Handoff(ctx context.Context, s scope, id string, qr application.QRContext) (domain.HandoffView, error) {
	out, err := scanConciergeHandoff(r.db(ctx).QueryRow(ctx, `
		SELECT `+conciergeHandoffColumns+`
		FROM concierge_handoffs h
		JOIN tables t
		  ON t.id=h.table_id AND t.organization_id=h.organization_id AND t.location_id=h.location_id
		LEFT JOIN users u ON u.id=h.resolved_by AND u.organization_id=h.organization_id
		WHERE h.id=$1 AND h.organization_id=$2 AND h.location_id=$3
	`, id, qr.Scope.OrganizationID, qr.Scope.LocationID))
	return out, conciergeError(err)
}
func (r *Concierge) HandoffStatus(ctx context.Context, s scope, conversationID string, qr application.QRContext) (domain.HandoffView, error) {
	out, err := scanConciergeHandoff(r.db(ctx).QueryRow(ctx, `
		SELECT `+conciergeHandoffColumns+`
		FROM concierge_handoffs h
		JOIN tables t
		  ON t.id=h.table_id AND t.organization_id=h.organization_id AND t.location_id=h.location_id
		LEFT JOIN users u ON u.id=h.resolved_by AND u.organization_id=h.organization_id
		WHERE h.organization_id=$1 AND h.location_id=$2 AND h.table_id=$3
		  AND h.conversation_id=$4
		ORDER BY h.requested_at DESC,h.id DESC
		LIMIT 1
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID, conversationID))
	return out, conciergeError(err)
}
func (r *Concierge) RequestHandoff(ctx context.Context, s scope, qr application.QRContext, in domain.HandoffInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO concierge_handoffs(
		  organization_id,location_id,table_id,conversation_id,customer_phone,reason
		)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(organization_id,location_id,conversation_id)
		  WHERE status='pending'
		DO UPDATE SET
		  customer_phone=EXCLUDED.customer_phone,
		  reason=EXCLUDED.reason
		RETURNING id::text
	`, qr.Scope.OrganizationID, qr.Scope.LocationID, qr.TableID, in.ConversationID, in.CustomerPhone, in.Reason).Scan(&id)
	return id, conciergeError(err)
}
func (r *Concierge) Handoffs(ctx context.Context, s scope, status string) ([]domain.HandoffView, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+conciergeHandoffColumns+`
		FROM concierge_handoffs h
		JOIN tables t
		  ON t.id=h.table_id AND t.organization_id=h.organization_id AND t.location_id=h.location_id
		LEFT JOIN users u ON u.id=h.resolved_by AND u.organization_id=h.organization_id
		WHERE h.organization_id=$1 AND h.location_id=$2 AND h.status=$3
		ORDER BY h.requested_at DESC,h.id DESC
		LIMIT 50
	`, s.OrganizationID, s.LocationID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.HandoffView{}
	for rows.Next() {
		var out domain.HandoffView
		if err := rows.Scan(
			&out.ID, &out.TableID, &out.TableName, &out.ConversationID, &out.CustomerPhone,
			&out.Reason, &out.Status, &out.RequestedAt, &out.ResolvedAt, &out.ResolvedByName,
		); err != nil {
			return nil, err
		}
		items = append(items, out)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, rows.Err()
}
func (r *Concierge) ResolveHandoff(ctx context.Context, s scope, id string) (string, error) {
	var resolvedID string
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE concierge_handoffs
		SET status='resolved',resolved_at=now(),resolved_by=$4
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status='pending'
		RETURNING id::text
	`, id, s.OrganizationID, s.LocationID, s.UserID).Scan(&resolvedID)
	return resolvedID, conciergeError(err)
}
