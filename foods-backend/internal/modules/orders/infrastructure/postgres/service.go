package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

func (r *Orders) LockServiceOrder(ctx context.Context, s scope, id string) (domain.Order, error) {
	o, err := scanOrder(r.db(ctx).QueryRow(ctx, `SELECT `+orderColumns+` FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`, id, s.OrganizationID, s.LocationID))
	return o, orderError(err)
}
func (r *Orders) CreateKitchenRound(ctx context.Context, s scope, orderID, status, source string) (string, int, error) {
	var sequence int
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(MAX(sequence),0)+1
		FROM order_kitchen_rounds
		WHERE order_id=$1 AND organization_id=$2 AND location_id=$3
	`, orderID, s.OrganizationID, s.LocationID).Scan(&sequence); err != nil {
		return "", 0, err
	}
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO order_kitchen_rounds(
		  organization_id,location_id,order_id,sequence,status,source
		)
		VALUES($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, s.OrganizationID, s.LocationID, orderID, sequence, status, source).Scan(&id)
	return id, sequence, err
}
func (r *Orders) AssignOrphanItems(ctx context.Context, s scope, orderID, roundID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE order_items
		SET kitchen_round_id=$3
		WHERE order_id=$1 AND organization_id=$2
		  AND kitchen_round_id IS NULL
	`, orderID, s.OrganizationID, roundID)
	return tag.RowsAffected(), err
}
func (r *Orders) UndeliveredItems(ctx context.Context, s scope, orderID string) (bool, error) {
	var pending bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2 AND status<>'entregado'
	)`, orderID, s.OrganizationID).Scan(&pending)
	return pending, err
}
func (r *Orders) CloseBill(ctx context.Context, s scope, o domain.Order) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE orders SET bill_closed_at=COALESCE(bill_closed_at,now()),bill_closed_by=COALESCE(bill_closed_by,CASE WHEN EXISTS(SELECT 1 FROM users WHERE id=$4 AND organization_id=$2) THEN $4::uuid END),updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, o.ID, s.OrganizationID, s.LocationID, s.UserID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) DeliverItem(ctx context.Context, s scope, o domain.Order, itemID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_service_items SET status='entregado',delivered_at=now(),delivered_by=CASE WHEN EXISTS(SELECT 1 FROM users WHERE id=$4 AND organization_id=$2) THEN $4::uuid END,updated_at=now() WHERE order_id=$1 AND organization_id=$2 AND id=$3 AND status='listo'`, o.ID, s.OrganizationID, itemID, s.UserID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) AppendAlreadyExists(ctx context.Context, s scope, o domain.Order, in application.AppendInput) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_kitchen_rounds WHERE organization_id=$1 AND order_id=$2 AND request_key=$3)`, s.OrganizationID, o.ID, in.RequestKey).Scan(&exists)
	return exists, orderError(err)
}
func (r *Orders) AccountPaid(ctx context.Context, s scope, o domain.Order) (bool, error) {
	var paid bool
	err := r.db(ctx).QueryRow(ctx, `SELECT `+netPaidSQL+` >= o.total AND (o.bill_closed_at IS NOT NULL OR `+netPaidSQL+` > 0) FROM orders o WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3`, o.ID, s.OrganizationID, s.LocationID).Scan(&paid)
	return paid, orderError(err)
}
func (r *Orders) IdentifyRound(ctx context.Context, s scope, roundID string, in application.AppendInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_kitchen_rounds SET request_key=$2 WHERE id=$1`, roundID, in.RequestKey)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) ReopenForAppend(ctx context.Context, s scope, o domain.Order) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE orders SET status='confirmado' WHERE id=$1 AND organization_id=$2`, o.ID, s.OrganizationID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) AddOrderTotal(ctx context.Context, s scope, o domain.Order, added float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE orders SET subtotal=subtotal+$4,total=total+$4,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, o.ID, s.OrganizationID, s.LocationID, added)
	return tag.RowsAffected(), orderError(err)
}
