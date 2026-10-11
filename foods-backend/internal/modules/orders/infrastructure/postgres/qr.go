package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

func (r *Orders) CreateQROrder(ctx context.Context, s scope, tableID string, in application.QROrderInput, subtotal float64) (domain.Order, error) {
	out, err := scanOrder(r.db(ctx).QueryRow(ctx, `
		INSERT INTO orders(
		  organization_id,location_id,channel,customer_name,customer_phone,
		  reference,table_id,notes,subtotal,delivery_fee,total,status,created_by
		)
		VALUES($1,$2,'whatsapp',$3,$4,'',$5,$6,$7,0,$7,'confirmado',NULL)
		RETURNING `+orderColumns,
		s.OrganizationID, s.LocationID, in.CustomerName, in.CustomerPhone,
		tableID, in.Notes, subtotal,
	))
	return out, orderError(err)
}
func (r *Orders) LockQRTable(ctx context.Context, s scope, tableID string, token string) (string, error) {
	var lockedTableID string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text
		FROM tables
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		  AND qr_token=$4 AND active AND qr_enabled
		FOR UPDATE
	`, tableID, s.OrganizationID, s.LocationID, token).Scan(&lockedTableID)
	return lockedTableID, orderError(err)
}
func (r *Orders) QRRequestOrder(ctx context.Context, s scope, tableID string, in application.QROrderInput) (string, error) {
	var existingOrderID string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT order_id::text
		FROM concierge_order_requests
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3 AND request_id=$4
	`, s.OrganizationID, s.LocationID, tableID, in.RequestID).Scan(&existingOrderID)
	return existingOrderID, orderError(err)
}
func (r *Orders) LockQRTableOrder(ctx context.Context, s scope, tableID string) (string, string, string, float64, float64, *string, bool, bool, error) {
	var openOrderID string
	var openStatus string
	var openChannel string
	var openSubtotal float64
	var openDeliveryFee float64
	var openCreatedBy *string
	var openBillClosed bool
	var conciergeOwned bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.id::text,o.status,o.channel,o.subtotal::float8,o.delivery_fee::float8,
		       o.created_by::text,o.bill_closed_at IS NOT NULL,
		       EXISTS(
		         SELECT 1 FROM audit_log a
		         WHERE a.organization_id=o.organization_id
		           AND a.entity_id=o.id
		           AND a.entity_type='order'
		           AND a.action='concierge.order.created'
		       )
		FROM orders o
		WHERE o.organization_id=$1 AND o.location_id=$2 AND o.table_id=$3
		  AND o.status<>'cancelado' AND (o.status<>'entregado' OR (o.channel='salon' AND o.completed_at IS NULL))
		ORDER BY o.created_at DESC,o.id
		LIMIT 1
		FOR UPDATE
	`, s.OrganizationID, s.LocationID, tableID).Scan(
		&openOrderID, &openStatus, &openChannel, &openSubtotal, &openDeliveryFee, &openCreatedBy, &openBillClosed, &conciergeOwned,
	)
	return openOrderID, openStatus, openChannel, openSubtotal, openDeliveryFee, openCreatedBy, openBillClosed, conciergeOwned, orderError(err)
}
func (r *Orders) QROrderTotal(ctx context.Context, s scope, openOrderID string) (float64, error) {
	var total float64
	err := r.db(ctx).QueryRow(ctx, `SELECT total::float8 FROM orders WHERE id=$1 AND organization_id=$2 AND location_id=$3`, openOrderID, s.OrganizationID, s.LocationID).Scan(&total)
	return total, orderError(err)
}
func (r *Orders) UpdateQRSubtotal(ctx context.Context, s scope, openOrderID string, nextSubtotal float64, nextTotal float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			UPDATE orders
			SET subtotal=$4,total=$5,updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
		`, openOrderID, s.OrganizationID, s.LocationID, nextSubtotal, nextTotal)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) SaveQRAppendRequest(ctx context.Context, s scope, tableID string, in application.QROrderInput, openOrderID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO concierge_order_requests(
			  organization_id,location_id,table_id,conversation_id,request_id,order_id
			)
			VALUES($1,$2,$3,$4,$5,$6)
		`, s.OrganizationID, s.LocationID, tableID, in.ConversationID, in.RequestID, openOrderID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) AuditQRAppend(ctx context.Context, s scope, tableID string, in application.QROrderInput, openOrderID string, openChannel string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO audit_log(
			  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
			)
			VALUES($1,$2,NULL,'concierge.order.items_added','order',$3,
			  jsonb_build_object(
			    'source','fudia_concierge',
			    'conversationId',$4::text,
			    'requestId',$5::text,
			    'tableId',$6::text,
			    'existingChannel',$7::text
			  ))
		`, s.OrganizationID, s.LocationID, openOrderID, in.ConversationID, in.RequestID, tableID, openChannel)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) SaveQRCreateRequest(ctx context.Context, s scope, tableID string, in application.QROrderInput, out domain.Order) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO concierge_order_requests(
		  organization_id,location_id,table_id,conversation_id,request_id,order_id
		)
		VALUES($1,$2,$3,$4,$5,$6)
	`, s.OrganizationID, s.LocationID, tableID, in.ConversationID, in.RequestID, out.ID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) AuditQRCreated(ctx context.Context, s scope, tableID string, in application.QROrderInput, out domain.Order) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO audit_log(
		  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
		)
		VALUES($1,$2,NULL,'concierge.order.created','order',$3,
		  jsonb_build_object('source','fudia_concierge','conversationId',$4::text,'requestId',$5::text,'tableId',$6::text))
	`, s.OrganizationID, s.LocationID, out.ID, in.ConversationID, in.RequestID, tableID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) OpenTableOrder(ctx context.Context, s scope, tableID string) (domain.Order, error) {
	out, err := scanOrder(r.db(ctx).QueryRow(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE organization_id=$1 AND location_id=$2 AND table_id=$3
		  AND status<>'cancelado' AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL))
		ORDER BY created_at DESC,id
		LIMIT 1
	`, s.OrganizationID, s.LocationID, tableID))
	return out, orderError(err)
}
