package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
)

func (r *Payments) CollectionShift(ctx context.Context, s scope) (*domain.CollectionShift, error) {
	var shift domain.CollectionShift
	err := r.db(ctx).QueryRow(ctx, `
  SELECT cs.id::text,cr.name,opened.full_name FROM cash_shift_users su
  JOIN cash_shifts cs ON cs.id=su.shift_id AND cs.organization_id=su.organization_id AND cs.location_id=su.location_id
  JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
  JOIN users opened ON opened.id=cs.opened_by AND opened.organization_id=cs.organization_id
  JOIN users u ON u.id=su.user_id AND u.organization_id=su.organization_id AND u.active
  WHERE su.organization_id=$1 AND su.location_id=$2 AND su.user_id=$3
   AND su.unassigned_at IS NULL AND cs.status='open'
   AND EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
    WHERE ur.user_id=$3 AND (ur.location_id IS NULL OR ur.location_id=$2 OR ro.system_key='administrator'))
  LIMIT 1`, s.OrganizationID, s.LocationID, s.UserID).Scan(&shift.ID, &shift.CashRegisterName, &shift.OpenedByName)
	return &shift, billingError(err)
}

func (r *Payments) CollectionOrderVisible(ctx context.Context, s scope, orderID, shiftID string) (bool, error) {
	var visible bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders o
  WHERE o.organization_id=$1 AND o.location_id=$2 AND o.id=$3 AND o.status<>'cancelado'
  AND o.channel='salon' AND o.table_id IS NOT NULL AND o.waiter_id=$5
  AND ((o.completed_at IS NULL AND o.total>(SELECT COALESCE(sum(p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=$1),0)),0) FROM payments p WHERE p.order_id=o.id AND p.organization_id=$1 AND p.location_id=$2))
   OR EXISTS(SELECT 1 FROM payments p WHERE p.order_id=o.id AND p.organization_id=$1 AND p.location_id=$2 AND p.shift_id=$4)))`, s.OrganizationID, s.LocationID, orderID, shiftID, s.UserID).Scan(&visible)
	return visible, err
}

func (r *Payments) OwnsCollectionOrder(ctx context.Context, s scope, id string) (bool, error) {
	var owned bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders
	 WHERE id=$1 AND organization_id=$2 AND location_id=$3
	 AND channel='salon' AND table_id IS NOT NULL AND waiter_id=$4)`, id, s.OrganizationID, s.LocationID, s.UserID).Scan(&owned)
	return owned, err
}

func (r *Payments) OpenCollectionShifts(ctx context.Context, s scope) ([]domain.CollectionShift, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT cs.id::text,cr.name,u.full_name FROM cash_shifts cs
 JOIN cash_registers cr ON cr.id=cs.cash_register_id AND cr.organization_id=cs.organization_id AND cr.location_id=cs.location_id
 JOIN users u ON u.id=cs.opened_by AND u.organization_id=cs.organization_id
 WHERE cs.organization_id=$1 AND cs.location_id=$2 AND cs.status='open'
 ORDER BY cr.name,cs.id`, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, billingError(err)
	}
	defer rows.Close()
	out := []domain.CollectionShift{}
	for rows.Next() {
		var item domain.CollectionShift
		if err = rows.Scan(&item.ID, &item.CashRegisterName, &item.OpenedByName); err != nil {
			return nil, billingError(err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func (r *Payments) LockCollectionShift(ctx context.Context, s scope, id string) (string, error) {
	var result string
	err := r.db(ctx).QueryRow(ctx, `SELECT id::text FROM cash_shifts WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND status='open' FOR SHARE`, id, s.OrganizationID, s.LocationID).Scan(&result)
	return result, billingError(err)
}
