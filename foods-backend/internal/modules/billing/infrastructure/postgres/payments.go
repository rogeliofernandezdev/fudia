package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scope = requestscope.Scope
type Payments struct{ pool *pgxpool.Pool }

func NewPayments(pool *pgxpool.Pool) *Payments { return &Payments{pool} }
func (r *Payments) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

var _ application.PaymentRepository = (*Payments)(nil)

func (r *Payments) CurrentShift(ctx context.Context, s scope) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT cs.id::text
		FROM cash_shift_users su
		JOIN cash_shifts cs ON cs.id=su.shift_id
		  AND cs.organization_id=su.organization_id
		  AND cs.location_id=su.location_id
		WHERE su.organization_id=$1
		  AND su.location_id=$2
		  AND su.user_id=$3
		  AND su.unassigned_at IS NULL
		  AND cs.status='open'
		ORDER BY su.assigned_at DESC
		LIMIT 1
	`, s.OrganizationID, s.LocationID, s.UserID).Scan(&id)
	return id, billingError(err)
}
func (r *Payments) ExpectedCash(ctx context.Context, s scope, shiftID string) (float64, error) {
	var expected float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT (
		  cs.opening_amount
		  + COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='income'),0)
		  - COALESCE((SELECT sum(cm.amount) FROM cash_movements cm WHERE cm.shift_id=cs.id AND cm.organization_id=cs.organization_id AND cm.movement_type='expense'),0)
		)::float8
		FROM cash_shifts cs
		WHERE cs.id=$1 AND cs.organization_id=$2 AND cs.location_id=$3 AND cs.status='open'
	`, shiftID, s.OrganizationID, s.LocationID).Scan(&expected)
	return expected, billingError(err)
}
func (r *Payments) RequireShiftAssignment(ctx context.Context, s scope, shiftID string) error {
	var assigned bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1
		  FROM cash_shift_users su
		  JOIN cash_shifts cs ON cs.id=su.shift_id
		    AND cs.organization_id=su.organization_id
		    AND cs.location_id=su.location_id
		  WHERE su.organization_id=$1
		    AND su.location_id=$2
		    AND su.shift_id=$3
		    AND su.user_id=$4
		    AND su.unassigned_at IS NULL
		    AND cs.status='open'
		)
	`, s.OrganizationID, s.LocationID, shiftID, s.UserID).Scan(&assigned)
	if err != nil {
		return err
	}
	if !assigned {
		return application.ErrNotFound
	}
	return nil
}
func (r *Payments) OrderPayments(ctx context.Context, s scope, orderID string) ([]domain.PaymentView, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id::text,p.order_id::text,o.code,p.shift_id::text,cr.name,p.method,pm.name,pm.affects_cash,p.amount::text,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::text,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0))::text,
		       p.reference,u.full_name,to_char(p.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		JOIN cash_shifts cs ON cs.id=p.shift_id AND cs.organization_id=p.organization_id AND cs.location_id=p.location_id
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		WHERE p.organization_id=$1 AND p.location_id=$2 AND p.order_id=$3
		ORDER BY p.created_at DESC,p.id DESC
	`, s.OrganizationID, s.LocationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PaymentView{}
	for rows.Next() {
		var item domain.PaymentView
		if err := rows.Scan(&item.ID, &item.OrderID, &item.OrderCode, &item.ShiftID, &item.CashRegisterName, &item.Method, &item.MethodName, &item.AffectsCash, &item.Amount, &item.RefundedAmount, &item.NetAmount, &item.Reference, &item.CreatedByName, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Payments) Payment(ctx context.Context, s scope, id string) (domain.PaymentView, error) {
	var item domain.PaymentView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.id::text,p.order_id::text,o.code,p.shift_id::text,cr.name,p.method,pm.name,pm.affects_cash,p.amount::text,
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::text,
		       (p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0))::text,
		       p.reference,u.full_name,to_char(p.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM payments p
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id
		JOIN cash_shifts cs ON cs.id=p.shift_id AND cs.organization_id=p.organization_id AND cs.location_id=p.location_id
		JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
		JOIN users u ON u.id=p.created_by AND u.organization_id=p.organization_id
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, id, s.OrganizationID, s.LocationID).Scan(
		&item.ID, &item.OrderID, &item.OrderCode, &item.ShiftID, &item.CashRegisterName, &item.Method, &item.MethodName, &item.AffectsCash, &item.Amount,
		&item.RefundedAmount, &item.NetAmount, &item.Reference, &item.CreatedByName, &item.CreatedAt,
	)
	return item, billingError(err)
}
func (r *Payments) LockPaymentOrder(ctx context.Context, s scope, in domain.PaymentInput) (string, string, float64, bool, bool, bool, error) {
	var orderCode string
	var status string
	var total float64
	var completed bool
	var salon bool
	var billClosed bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.code,o.status,o.total::float8,o.completed_at IS NOT NULL,o.channel='salon',o.bill_closed_at IS NOT NULL
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&orderCode, &status, &total, &completed, &salon, &billClosed)
	return orderCode, status, total, completed, salon, billClosed, billingError(err)
}
func (r *Payments) PaidForOrder(ctx context.Context, s scope, in domain.PaymentInput) (float64, error) {
	var paid float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&paid)
	return paid, billingError(err)
}
func (r *Payments) CreatePayment(ctx context.Context, s scope, in domain.PaymentInput, shiftID string) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,reference,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text
	`, s.OrganizationID, s.LocationID, in.OrderID, shiftID, in.Method, in.Amount, in.Reference, s.UserID).Scan(&id)
	return id, billingError(err)
}
func (r *Payments) SaleMovement(ctx context.Context, s scope, in domain.PaymentInput, shiftID string, id string, orderCode string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'income','cash_sale',$4::uuid,$5,$6,$7,$8)
		`, s.OrganizationID, s.LocationID, shiftID, id, in.Amount, "Cobro "+orderCode, in.Reference, s.UserID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) LockRefundPayment(ctx context.Context, s scope, id string) (bool, string, string, float64, bool, bool, float64, error) {
	var affectsCash bool
	var orderCode string
	var orderStatus string
	var amount float64
	var completed bool
	var salon bool
	var refunded float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT pm.affects_cash,o.code,o.status,p.amount::float8,o.completed_at IS NOT NULL,o.channel='salon',
		       COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)::float8
		FROM payments p
		JOIN payment_methods pm ON pm.organization_id=p.organization_id AND pm.code=p.method
		JOIN orders o ON o.id=p.order_id AND o.organization_id=p.organization_id AND o.location_id=p.location_id
		WHERE p.id=$1 AND p.organization_id=$2 AND p.location_id=$3
		FOR UPDATE
	`, id, s.OrganizationID, s.LocationID).Scan(&affectsCash, &orderCode, &orderStatus, &amount, &completed, &salon, &refunded)
	return affectsCash, orderCode, orderStatus, amount, completed, salon, refunded, billingError(err)
}
func (r *Payments) CreateRefund(ctx context.Context, s scope, id string, in domain.RefundInput, shiftID string) (string, error) {
	var refundID string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO payment_refunds(organization_id,location_id,payment_id,shift_id,amount,reason,note,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text
	`, s.OrganizationID, s.LocationID, id, shiftID, in.Amount, in.Reason, in.Note, s.UserID).Scan(&refundID)
	return refundID, billingError(err)
}
func (r *Payments) RefundMovement(ctx context.Context, s scope, in domain.RefundInput, shiftID string, refundID string, orderCode string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO cash_movements(
			  organization_id,location_id,shift_id,movement_type,source_type,source_id,
			  amount,reason,note,created_by
			)
			VALUES($1,$2,$3,'expense','cash_refund',$4::uuid,$5,$6,$7,$8)
		`, s.OrganizationID, s.LocationID, shiftID, refundID, in.Amount, "Devolución "+orderCode, in.Note, s.UserID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) LockBatchOrder(ctx context.Context, s scope, in domain.PaymentBatchInput) (string, string, float64, bool, bool, bool, error) {
	var orderCode string
	var status string
	var orderTotal float64
	var completed bool
	var salon bool
	var billClosed bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.code,o.status,o.total::float8,o.completed_at IS NOT NULL,o.channel='salon',o.bill_closed_at IS NOT NULL
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&orderCode, &status, &orderTotal, &completed, &salon, &billClosed)
	return orderCode, status, orderTotal, completed, salon, billClosed, billingError(err)
}
func (r *Payments) BatchPaid(ctx context.Context, s scope, in domain.PaymentBatchInput) (float64, error) {
	var paid float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, in.OrderID, s.OrganizationID, s.LocationID).Scan(&paid)
	return paid, billingError(err)
}
func (r *Payments) CreateBatchPayment(ctx context.Context, s scope, in domain.PaymentBatchInput, shiftID string, p domain.PaymentLine) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO payments(organization_id,location_id,order_id,shift_id,method,amount,reference,created_by)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id::text
		`, s.OrganizationID, s.LocationID, in.OrderID, shiftID, p.Method, p.Amount, p.Reference, s.UserID).Scan(&id)
	return id, billingError(err)
}
func (r *Payments) BatchSaleMovement(ctx context.Context, s scope, shiftID string, id string, p domain.PaymentLine, orderCode string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
				INSERT INTO cash_movements(
				  organization_id,location_id,shift_id,movement_type,source_type,source_id,
				  amount,reason,note,created_by
				)
				VALUES($1,$2,$3,'income','cash_sale',$4::uuid,$5,$6,$7,$8)
			`, s.OrganizationID, s.LocationID, shiftID, id, p.Amount, "Cobro "+orderCode, p.Reference, s.UserID)
	return tag.RowsAffected(), billingError(err)
}
func (r *Payments) POSOrders(ctx context.Context, s scope, f application.POSFilter) (application.POSPage, error) {
	out := application.POSPage{Page: f.Page, PageSize: f.Size}
	page, size, q, from, to, status := f.Page, f.Size, f.Query, f.From, f.To, f.Status
	filter := ""
	switch status {
	case "unpaid":
		filter = " AND payment_summary.net_paid < o.total"
	case "pending":
		filter = " AND payment_summary.net_paid <= 0.00001"
	case "partial":
		filter = " AND payment_summary.net_paid > 0.00001 AND payment_summary.net_paid < o.total"
	case "paid":
		filter = " AND payment_summary.net_paid >= o.total"
	}

	baseFrom := `
		FROM orders o
		JOIN locations loc ON loc.id=o.location_id AND loc.organization_id=o.organization_id
		LEFT JOIN tables t
		  ON t.id=o.table_id
		 AND t.organization_id=o.organization_id
		 AND t.location_id=o.location_id
		LEFT JOIN LATERAL (
		  SELECT COALESCE(sum(p.amount-COALESCE(refunds.refunded,0)),0) AS net_paid,
		         COALESCE(array_agg(DISTINCT pm.name ORDER BY pm.name)
		           FILTER (WHERE p.amount-COALESCE(refunds.refunded,0)>0 AND pm.name IS NOT NULL),
		           '{}'::text[]) AS payment_methods
		  FROM payments p
		  LEFT JOIN payment_methods pm
		    ON pm.organization_id=p.organization_id AND pm.code=p.method
		  LEFT JOIN LATERAL (
		    SELECT COALESCE(sum(pr.amount),0) AS refunded
		    FROM payment_refunds pr
		    WHERE pr.payment_id=p.id
		      AND pr.organization_id=p.organization_id
		      AND pr.location_id=p.location_id
		  ) refunds ON true
		  WHERE p.order_id=o.id
		    AND p.organization_id=o.organization_id
		    AND p.location_id=o.location_id
		) payment_summary ON true
	`
	where := `
		o.organization_id=$1 AND o.location_id=$2 AND o.status<>'cancelado'
		AND ($3='' OR o.code ILIKE '%'||$3||'%' OR o.customer_name ILIKE '%'||$3||'%'
		  OR COALESCE(t.name,'') ILIKE '%'||$3||'%')
		AND ($4='' OR o.created_at >= (NULLIF($4,'')::date::timestamp AT TIME ZONE loc.timezone))
		AND ($5='' OR o.created_at < ((NULLIF($5,'')::date + interval '1 day') AT TIME ZONE loc.timezone))
	` + filter

	var totalCount int
	if err := r.db(ctx).QueryRow(ctx, "SELECT count(*),COALESCE(sum(payment_summary.net_paid),0)::text "+baseFrom+" WHERE "+where, s.OrganizationID, s.LocationID, q, from, to).Scan(&totalCount, &out.AmountTotal); err != nil {
		return out, err
	}

	rows, err := r.db(ctx).Query(ctx, `
		SELECT o.id::text,o.code,o.channel,o.status,o.customer_name,
		       COALESCE(t.name,''),
		       o.total::text,
		       payment_summary.net_paid::text,
		       GREATEST(o.total-payment_summary.net_paid,0)::text,
		       CASE
		         WHEN payment_summary.net_paid <= 0.00001 THEN 'pending'
		         WHEN payment_summary.net_paid >= o.total THEN 'paid'
		         ELSE 'partial'
		       END,
		       to_char(o.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       payment_summary.payment_methods,o.bill_closed_at IS NOT NULL
	`+baseFrom+`
		WHERE `+where+`
		ORDER BY o.created_at DESC,o.id DESC
		LIMIT $6 OFFSET $7
	`, s.OrganizationID, s.LocationID, q, from, to, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	items := []domain.PosOrderSummary{}
	for rows.Next() {
		var item domain.PosOrderSummary
		if err := rows.Scan(
			&item.ID, &item.Code, &item.Channel, &item.Status, &item.CustomerName, &item.TableName,
			&item.Total, &item.PaidAmount, &item.RemainingAmount, &item.PaymentStatus, &item.CreatedAt,
			&item.PaymentMethods, &item.BillClosed,
		); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Items = items
	out.Total = totalCount
	return out, rows.Err()
}
func (r *Payments) ReceiptContext(ctx context.Context, s scope) (domain.PaymentReceiptContext, error) {
	var identity domain.PaymentReceiptContext
	err := r.db(ctx).QueryRow(ctx, `SELECT COALESCE(NULLIF(org.trade_name,''),org.legal_name),org.legal_name,org.tax_id,
 loc.name,loc.address,loc.phone,fp.country_code,loc.timezone,fp.currency,fp.currency_symbol,fp.currency_position,fp.currency_decimals
 FROM locations loc JOIN organizations org ON org.id=loc.organization_id
 JOIN organization_fiscal_profiles fp ON fp.id=loc.fiscal_profile_id AND fp.organization_id=loc.organization_id
 WHERE loc.id=$1 AND loc.organization_id=$2`, s.LocationID, s.OrganizationID).Scan(
		&identity.OrganizationName, &identity.LegalName, &identity.TaxID, &identity.LocationName, &identity.Address, &identity.Phone,
		&identity.Country, &identity.Timezone, &identity.Currency, &identity.CurrencySymbol, &identity.CurrencyPosition, &identity.CurrencyDecimals)
	return identity, billingError(err)
}
