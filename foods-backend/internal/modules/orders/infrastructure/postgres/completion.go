package postgres

import "context"

func (r *Orders) LockPaidCompletion(ctx context.Context, s scope, id string) (float64, string, bool, error) {
	var total float64
	var status string
	var salon bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.total::float8,o.status,o.channel='salon'
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		FOR UPDATE
	`, id, s.OrganizationID, s.LocationID).Scan(&total, &status, &salon)
	return total, status, salon, orderError(err)
}
func (r *Orders) CompletionPaid(ctx context.Context, s scope, id string) (float64, error) {
	var paid float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(sum(
		  p.amount-COALESCE((SELECT sum(pr.amount) FROM payment_refunds pr WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id),0)
		),0)::float8
		FROM payments p
		WHERE p.order_id=$1 AND p.organization_id=$2 AND p.location_id=$3
	`, id, s.OrganizationID, s.LocationID).Scan(&paid)
	return paid, orderError(err)
}
func (r *Orders) CompletionPending(ctx context.Context, s scope, id string) (bool, error) {
	var pending bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2 AND status NOT IN ('listo','entregado'))`, id, s.OrganizationID).Scan(&pending)
	return pending, orderError(err)
}
func (r *Orders) DeliverPaidItems(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_service_items SET status='entregado',delivered_at=now(),delivered_by=(SELECT id FROM users WHERE id=$3 AND organization_id=$2),updated_at=now() WHERE order_id=$1 AND organization_id=$2 AND status='listo'`, id, s.OrganizationID, s.UserID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) CompletePaidOrder(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE orders
		SET status='entregado',updated_at=now(),completed_at=COALESCE(completed_at,now())
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, id, s.OrganizationID, s.LocationID)
	return tag.RowsAffected(), orderError(err)
}
