package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

// LockConciergeDeliveryRequest serializa reintentos concurrentes del mismo requestId.
func (r *Orders) LockConciergeDeliveryRequest(ctx context.Context, s scope, requestID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('concierge_delivery|'||$1::text||'|'||$2::text||'|'||$3::text,0))
	`, s.OrganizationID, s.LocationID, requestID)
	return orderError(err)
}
func (r *Orders) ConciergeDeliveryRequestOrder(ctx context.Context, s scope, requestID string) (string, error) {
	var orderID string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT order_id::text
		FROM concierge_delivery_requests
		WHERE organization_id=$1 AND location_id=$2 AND request_id=$3
	`, s.OrganizationID, s.LocationID, requestID).Scan(&orderID)
	return orderID, orderError(err)
}
func (r *Orders) CreateConciergeDeliveryOrder(ctx context.Context, s scope, in application.ConciergeDeliveryInput, subtotal float64, total float64, customerID string) (domain.Order, error) {
	out, err := scanOrder(r.db(ctx).QueryRow(ctx, `
		INSERT INTO orders(
		  organization_id,location_id,channel,customer_id,customer_name,customer_phone,address,reference,
		  table_id,notes,subtotal,delivery_fee,total,status,created_by,
		  delivery_latitude,delivery_longitude,delivery_distance_km
		)
		VALUES($1,$2,$3,NULLIF($15,'')::uuid,$4,$5,$6,$7,NULL,$8,$9,$10,$11,'confirmado',NULL,$12,$13,$14)
		RETURNING `+orderColumns,
		s.OrganizationID, s.LocationID, in.Channel, in.CustomerName, in.CustomerPhone, in.Address, in.Reference,
		in.Notes, subtotal, in.DeliveryFee, total, in.Latitude, in.Longitude, in.DistanceKm, customerID,
	))
	return out, orderError(err)
}
func (r *Orders) SaveConciergeDeliveryRequest(ctx context.Context, s scope, in application.ConciergeDeliveryInput, orderID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO concierge_delivery_requests(organization_id,location_id,request_id,conversation_id,order_id)
		VALUES($1,$2,$3,$4,$5)
	`, s.OrganizationID, s.LocationID, in.RequestID, in.ConversationID, orderID)
	return orderError(err)
}
func (r *Orders) AuditConciergeDeliveryCreated(ctx context.Context, s scope, in application.ConciergeDeliveryInput, orderID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO audit_log(
		  organization_id,location_id,user_id,action,entity_type,entity_id,metadata
		)
		VALUES($1,$2,NULL,'concierge.delivery_order.created','order',$3,
		  jsonb_build_object(
		    'source','fudia_concierge',
		    'conversationId',$4::text,
		    'requestId',$5::text,
		    'channel',$6::text,
		    'deliveryFee',$7::numeric,
		    'distanceKm',$8::numeric
		  ))
	`, s.OrganizationID, s.LocationID, orderID, in.ConversationID, in.RequestID, in.Channel, in.DeliveryFee, in.DistanceKm)
	return orderError(err)
}

// ConciergeOrderActive usa el mismo criterio de "comanda abierta" que la mesa:
// un delivery o recojo deja de estar activo al entregarse o cancelarse.
func (r *Orders) ConciergeOrderActive(ctx context.Context, s scope, conversationID string, orderID string) (bool, error) {
	var active bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.status<>'cancelado' AND (o.status<>'entregado' OR (o.channel='salon' AND o.completed_at IS NULL))
		FROM orders o
		WHERE o.id::text=$3 AND o.organization_id=$1 AND o.location_id=$2
		  AND (
		    EXISTS(
		      SELECT 1 FROM concierge_order_requests q
		      WHERE q.organization_id=o.organization_id AND q.location_id=o.location_id
		        AND q.order_id=o.id AND q.conversation_id=$4
		    )
		    OR EXISTS(
		      SELECT 1 FROM concierge_delivery_requests d
		      WHERE d.organization_id=o.organization_id AND d.location_id=o.location_id
		        AND d.order_id=o.id AND d.conversation_id=$4
		    )
		  )
	`, s.OrganizationID, s.LocationID, orderID, conversationID).Scan(&active)
	return active, orderError(err)
}
