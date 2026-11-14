package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/application"
	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/jackc/pgx/v5"
)

var _ application.PanelRepository = (*Repository)(nil)

func (r *Repository) PartnerByPanelToken(ctx context.Context, token string) (application.PanelPartner, error) {
	var p application.PanelPartner
	err := r.db(ctx).QueryRow(ctx, `SELECT id::text,name,active FROM delivery_partners WHERE panel_token=$1`, token).Scan(&p.ID, &p.Name, &p.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, application.ErrNotFound
	}
	return p, err
}

// StartOfferTimers inicia los 3 minutos de las ofertas pendientes que el panel
// muestra por primera vez; una oferta ya enviada por WhatsApp conserva su reloj.
func (r *Repository) StartOfferTimers(ctx context.Context, partnerID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatch_offers SET expires_at=now()+make_interval(secs => $2)
		WHERE partner_id=$1 AND status='pending' AND expires_at IS NULL`, partnerID, domain.OfferTimeout.Seconds())
	return err
}

// PanelOrders lista, para una central: las ofertas vigentes que aún esperan su
// respuesta, sus repartos en curso y los entregados en las últimas horas.
func (r *Repository) PanelOrders(ctx context.Context, partnerID string) ([]domain.PanelOrder, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT f.action_token::text,
		       CASE WHEN f.status='pending' THEN 'offer'
		            WHEN d.status='delivered' OR o.status='entregado' THEN 'delivered'
		            WHEN d.status='picked_up' THEN 'picked_up'
		            ELSE 'accepted' END AS stage,
		       o.code,o.status,COALESCE(NULLIF(btrim(org.trade_name),''),org.legal_name),l.name,l.address,
		       l.latitude::float8,l.longitude::float8,
		       o.customer_name,o.customer_phone,o.address,o.reference,
		       o.delivery_latitude::float8,o.delivery_longitude::float8,o.delivery_distance_km::float8,o.notes,
		       COALESCE((
		         SELECT json_agg(json_build_object('quantity',i.qty::float8,'name',i.name,'note',i.note) ORDER BY i.created_at,i.id)
		         FROM order_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id
		       ),'[]'::json)::text,
		       COALESCE(fp.currency_symbol,''),o.subtotal::text,
		       COALESCE(d.partner_fee,o.delivery_fee)::text,(o.subtotal+COALESCE(d.partner_fee,o.delivery_fee))::text,
		       CASE WHEN f.status='pending' THEN GREATEST(0,ceil(EXTRACT(EPOCH FROM f.expires_at-now())))::int ELSE 0 END,
		       f.created_at,d.accepted_at,d.picked_up_at,d.delivered_at
		FROM delivery_dispatch_offers f
		JOIN delivery_partners p ON p.id=f.partner_id
		JOIN delivery_dispatches d ON d.id=f.dispatch_id
		JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		JOIN organizations org ON org.id=d.organization_id
		JOIN locations l ON l.id=d.location_id AND l.organization_id=d.organization_id
		LEFT JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
		WHERE f.partner_id=$1 AND f.round=d.round AND (
		  (f.status='pending' AND d.status='searching' AND f.expires_at>now() AND p.active AND `+partnerCoverageSQL+`
		     AND o.paid_amount=0 AND o.completed_at IS NULL AND o.status IN ('confirmado','preparando','listo'))
		  OR (f.status='accepted' AND d.accepted_partner_id=f.partner_id AND (
		       (d.status IN ('accepted','picked_up') AND o.status<>'cancelado'
		          AND (o.status<>'entregado' OR d.updated_at>now()-make_interval(hours => $2)))
		       OR (d.status='delivered' AND d.delivered_at>now()-make_interval(hours => $2)))))
		ORDER BY CASE WHEN f.status='pending' THEN 0 WHEN d.status='delivered' OR o.status='entregado' THEN 2 ELSE 1 END,
		         f.created_at DESC,f.id`, partnerID, domain.PanelHistoryHours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PanelOrder{}
	for rows.Next() {
		var o domain.PanelOrder
		var items string
		var created time.Time
		var accepted, picked, delivered *time.Time
		if err := rows.Scan(&o.Token, &o.Stage, &o.OrderCode, &o.OrderStatus, &o.OrganizationName, &o.LocationName, &o.LocationAddress,
			&o.LocationLatitude, &o.LocationLongitude,
			&o.CustomerName, &o.CustomerPhone, &o.Address, &o.Reference,
			&o.Latitude, &o.Longitude, &o.DistanceKm, &o.Notes, &items,
			&o.CurrencySymbol, &o.Subtotal, &o.DeliveryFee, &o.Total,
			&o.ExpiresInSeconds, &created, &accepted, &picked, &delivered); err != nil {
			return nil, err
		}
		o.Items = []domain.MessageItem{}
		if err := json.Unmarshal([]byte(items), &o.Items); err != nil {
			return nil, err
		}
		o.CreatedAt = stamp(created)
		o.AcceptedAt, o.PickedUpAt, o.DeliveredAt = optionalStamp(accepted), optionalStamp(picked), optionalStamp(delivered)
		out = append(out, o)
	}
	return out, rows.Err()
}

// SetCourierLocation guarda la ubicación en los repartos que la central tiene
// en curso. Un reparto entregado o cancelado ya no la recibe.
func (r *Repository) SetCourierLocation(ctx context.Context, partnerID string, latitude, longitude float64) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches SET courier_latitude=$2,courier_longitude=$3,courier_located_at=now()
		WHERE accepted_partner_id=$1 AND status IN ('accepted','picked_up')`, partnerID, latitude, longitude)
	return err
}

// Tracking arma el seguimiento del pedido de delivery. La ubicación del
// repartidor solo sale mientras el reparto sigue en curso y es reciente.
func (r *Repository) Tracking(ctx context.Context, s scope, orderID string) (domain.Tracking, error) {
	var out domain.Tracking
	var originLat, originLng, destLat, destLng, courierLat, courierLng *float64
	var located, accepted, picked, delivered *time.Time
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(d.status,''),COALESCE(p.name,''),l.name,l.address,
		       l.latitude::float8,l.longitude::float8,o.address,
		       o.delivery_latitude::float8,o.delivery_longitude::float8,
		       CASE WHEN live.ok THEN d.courier_latitude::float8 END,
		       CASE WHEN live.ok THEN d.courier_longitude::float8 END,
		       CASE WHEN live.ok THEN d.courier_located_at END,
		       d.accepted_at,d.picked_up_at,d.delivered_at
		FROM orders o
		JOIN locations l ON l.id=o.location_id AND l.organization_id=o.organization_id
		LEFT JOIN delivery_dispatches d ON d.order_id=o.id AND d.organization_id=o.organization_id
		LEFT JOIN delivery_partners p ON p.id=d.accepted_partner_id AND d.status IN ('accepted','picked_up','delivered')
		LEFT JOIN LATERAL (
		  SELECT COALESCE(d.status IN ('accepted','picked_up') AND o.status NOT IN ('entregado','cancelado')
		    AND d.courier_located_at>now()-make_interval(mins => $4),false) AS ok
		) live ON true
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3 AND o.channel='delivery'`,
		orderID, s.OrganizationID, s.LocationID, domain.CourierLocationMaxAgeMinutes).Scan(
		&out.DispatchStatus, &out.PartnerName, &out.OriginName, &out.OriginAddress,
		&originLat, &originLng, &out.DestinationAddress, &destLat, &destLng,
		&courierLat, &courierLng, &located, &accepted, &picked, &delivered)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	out.Origin, out.Destination, out.Courier = point(originLat, originLng), point(destLat, destLng), point(courierLat, courierLng)
	out.CourierLocatedAt = optionalStamp(located)
	out.AcceptedAt, out.PickedUpAt, out.DeliveredAt = optionalStamp(accepted), optionalStamp(picked), optionalStamp(delivered)
	return out, nil
}

func point(latitude, longitude *float64) *domain.Point {
	if latitude == nil || longitude == nil {
		return nil
	}
	return &domain.Point{Latitude: *latitude, Longitude: *longitude}
}
