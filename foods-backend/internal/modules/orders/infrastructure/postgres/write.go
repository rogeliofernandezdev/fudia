package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"strings"
)

func (r *Orders) CreateOrder(ctx context.Context, s scope, in domain.OrderInput, customerID *string, tableID *string, subtotal float64, total float64, initialStatus string) (domain.Order, error) {
	o, err := scanOrder(r.db(ctx).QueryRow(ctx, `
		INSERT INTO orders(organization_id,location_id,channel,customer_id,customer_name,customer_phone,address,reference,table_id,notes,subtotal,delivery_fee,total,status,created_by,waiter_id,delivery_latitude,delivery_longitude,delivery_distance_km)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,CASE WHEN $3='salon' AND $9::uuid IS NOT NULL AND EXISTS(SELECT 1 FROM users u WHERE u.id=$15::uuid AND u.organization_id=$1::uuid) THEN $15::uuid ELSE NULL END,$16,$17,$18)
		RETURNING `+orderColumns,
		s.OrganizationID, s.LocationID, in.Channel, customerID, in.CustomerName,
		strings.TrimSpace(in.CustomerPhone), strings.TrimSpace(in.Address), strings.TrimSpace(in.Reference),
		tableID, strings.TrimSpace(in.Notes), subtotal, in.DeliveryFee, total, initialStatus, s.UserID,
		in.DeliveryLatitude, in.DeliveryLongitude, in.DeliveryDistanceKm))
	return o, orderError(err)
}
func (r *Orders) LocationPoint(ctx context.Context, s scope) (*float64, *float64, error) {
	var latitude, longitude *float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT latitude::float8,longitude::float8
		FROM locations
		WHERE id=$2 AND organization_id=$1
	`, s.OrganizationID, s.LocationID).Scan(&latitude, &longitude)
	return latitude, longitude, orderError(err)
}
func (r *Orders) CustomerActive(ctx context.Context, s scope, in domain.OrderInput) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE id=$1 AND organization_id=$2 AND active)`, in.CustomerID, s.OrganizationID).Scan(&ok)
	return ok, orderError(err)
}
func (r *Orders) LockTable(ctx context.Context, s scope, in domain.OrderInput) (string, error) {
	var lockedTableID string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT id
			FROM tables
			WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND active
			FOR UPDATE`, in.TableID, s.OrganizationID, s.LocationID).Scan(&lockedTableID)
	return lockedTableID, orderError(err)
}
func (r *Orders) TableOccupied(ctx context.Context, s scope, in domain.OrderInput) (bool, error) {
	var occupied bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders WHERE organization_id=$1 AND location_id=$2 AND table_id=$3 AND status<>'cancelado' AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL)))`, s.OrganizationID, s.LocationID, in.TableID).Scan(&occupied)
	return occupied, orderError(err)
}
func (r *Orders) LockEditableOrder(ctx context.Context, s scope, id string) (string, string, string, error) {
	var currentStatus string
	var channel string
	var waiterID string
	err := r.db(ctx).QueryRow(ctx, `SELECT status,channel,COALESCE(waiter_id::text,'') FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`, id, s.OrganizationID, s.LocationID).Scan(&currentStatus, &channel, &waiterID)
	return currentStatus, channel, waiterID, orderError(err)
}
func (r *Orders) HasServiceItems(ctx context.Context, s scope, id string) (bool, error) {
	var sent bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2)`, id, s.OrganizationID).Scan(&sent)
	return sent, orderError(err)
}
func (r *Orders) HasKitchenRounds(ctx context.Context, s scope, id string) (bool, error) {
	var hasKitchenRounds bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_kitchen_rounds WHERE order_id=$1 AND organization_id=$2 AND location_id=$3)`, id, s.OrganizationID, s.LocationID).Scan(&hasKitchenRounds)
	return hasKitchenRounds, orderError(err)
}
func (r *Orders) UpdateOrder(ctx context.Context, s scope, id string, in domain.OrderUpdateInput, subtotal float64, total float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE orders
		SET customer_name=$4,customer_phone=$5,address=$6,reference=$7,notes=$8,subtotal=$9,delivery_fee=$10,total=$11,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		id, s.OrganizationID, s.LocationID,
		in.CustomerName, strings.TrimSpace(in.CustomerPhone), strings.TrimSpace(in.Address), strings.TrimSpace(in.Reference), strings.TrimSpace(in.Notes),
		subtotal, in.DeliveryFee, total)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) DeleteItems(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM order_items WHERE order_id=$1 AND organization_id=$2`, id, s.OrganizationID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) LockOrderStatus(ctx context.Context, s scope, id string) (string, string, string, float64, bool, string, error) {
	var current string
	var channel string
	var tableID string
	var total float64
	var completed bool
	var waiterID string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT status,channel,COALESCE(table_id::text,''),total::float8,completed_at IS NOT NULL,COALESCE(waiter_id::text,'')
		FROM orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE`, id, s.OrganizationID, s.LocationID).Scan(&current, &channel, &tableID, &total, &completed, &waiterID)
	return current, channel, tableID, total, completed, waiterID, orderError(err)
}
func (r *Orders) ServiceStarted(ctx context.Context, s scope, id string) (bool, bool, error) {
	var hasService bool
	var started bool
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*)>0,
			COALESCE(bool_or(status='entregado' OR (destination<>'direct' AND status IN ('preparando','listo'))),false)
			FROM order_service_items WHERE order_id=$1 AND organization_id=$2`,
		id, s.OrganizationID).Scan(&hasService, &started)
	return hasService, started, orderError(err)
}
func (r *Orders) SubmitServiceItems(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_service_items SET status=CASE WHEN destination='direct' THEN 'listo' ELSE 'confirmado' END,updated_at=now() WHERE order_id=$1 AND organization_id=$2 AND status='nuevo'`, id, s.OrganizationID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) PendingServiceItems(ctx context.Context, s scope, id string) (bool, error) {
	var pending bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2 AND status NOT IN ('listo','entregado'))`, id, s.OrganizationID).Scan(&pending)
	return pending, orderError(err)
}
func (r *Orders) DeliverServiceItems(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_service_items SET status='entregado',delivered_at=now(),delivered_by=(SELECT id FROM users WHERE id=$3 AND organization_id=$2),updated_at=now() WHERE order_id=$1 AND organization_id=$2 AND status='listo'`, id, s.OrganizationID, s.UserID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) SetStatus(ctx context.Context, s scope, id string, in application.OrderStatusInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE orders
		SET status=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		id, s.OrganizationID, s.LocationID, in.Status)
	return tag.RowsAffected(), orderError(err)
}
