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

func (r *Repository) OrderFacts(ctx context.Context, s scope, orderID string) (application.OrderFacts, error) {
	var f application.OrderFacts
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.id::text,o.code,o.channel,o.status,o.customer_name,o.completed_at IS NOT NULL,
		       CASE WHEN COALESCE(ds.enabled,false) THEN 'fudia' ELSE 'own' END,o.paid_amount>0
		FROM orders o
		LEFT JOIN organization_delivery_settings ds ON ds.organization_id=o.organization_id
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3 FOR UPDATE OF o`,
		orderID, s.OrganizationID, s.LocationID).Scan(&f.ID, &f.Code, &f.Channel, &f.Status, &f.CustomerName, &f.Completed, &f.Mode, &f.HasPayments)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, application.ErrNotFound
	}
	return f, err
}

const dispatchStateColumns = `id::text,order_id::text,status,round,ready_notified_at IS NOT NULL`

func scanState(row pgx.Row) (application.DispatchState, error) {
	var d application.DispatchState
	err := row.Scan(&d.ID, &d.OrderID, &d.Status, &d.Round, &d.ReadyNotified)
	return d, err
}

func (r *Repository) LockDispatch(ctx context.Context, s scope, orderID string) (application.DispatchState, bool, error) {
	d, err := scanState(r.db(ctx).QueryRow(ctx, `
		SELECT `+dispatchStateColumns+` FROM delivery_dispatches
		WHERE order_id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`,
		orderID, s.OrganizationID, s.LocationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, false, nil
	}
	return d, err == nil, err
}

func (r *Repository) CreateDispatch(ctx context.Context, s scope, orderID, status string) (application.DispatchState, bool, error) {
	d, err := scanState(r.db(ctx).QueryRow(ctx, `
		INSERT INTO delivery_dispatches(organization_id,location_id,order_id,status) VALUES ($1,$2,$3,$4)
		ON CONFLICT (order_id) DO NOTHING
		RETURNING `+dispatchStateColumns, s.OrganizationID, s.LocationID, orderID, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, false, nil
	}
	return d, err == nil, err
}

func (r *Repository) SetDispatchStatus(ctx context.Context, s scope, dispatchID, status string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches SET status=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`, dispatchID, s.OrganizationID, s.LocationID, status)
	return err
}

func (r *Repository) StartRound(ctx context.Context, s scope, dispatchID string, round int) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches
		SET round=$4,status='searching',accepted_partner_id=NULL,accepted_at=NULL,ready_notified_at=NULL,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`, dispatchID, s.OrganizationID, s.LocationID, round)
	return err
}

// The same predicate governs initial selection, pending messages and buttons.
const partnerCoverageSQL = `EXISTS (SELECT 1 FROM delivery_partner_coverage c
 WHERE c.partner_id=p.id AND l.district_code=c.area_code)
 AND EXISTS (SELECT 1 FROM organization_fiscal_profiles fp WHERE fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id AND fp.country_code='PE')`

func (r *Repository) HasActivePartners(ctx context.Context, s scope) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM delivery_partners p CROSS JOIN locations l
        WHERE p.active AND l.organization_id=$1 AND l.id=$2 AND `+partnerCoverageSQL+`)`, s.OrganizationID, s.LocationID).Scan(&ok)
	return ok, err
}

// NextPartner toma la siguiente central activa del orden global que aún no se
// ofreció en la vuelta: un rechazo o un vencimiento nunca vuelve a la misma.
func (r *Repository) NextPartner(ctx context.Context, s scope, dispatchID string, round int) (string, int, bool, error) {
	var partnerID string
	var priority int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.id::text,
		       (SELECT COALESCE(max(f.priority),0)+1 FROM delivery_dispatch_offers f WHERE f.dispatch_id=$1 AND f.round=$2)
		FROM delivery_partners p CROSS JOIN locations l
		WHERE p.active AND l.organization_id=$3 AND l.id=$4 AND `+partnerCoverageSQL+` AND NOT EXISTS (
		  SELECT 1 FROM delivery_dispatch_offers f
		  WHERE f.dispatch_id=$1 AND f.round=$2 AND f.partner_id=p.id)
		ORDER BY p.priority,p.id
		LIMIT 1`, dispatchID, round, s.OrganizationID, s.LocationID).Scan(&partnerID, &priority)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, false, nil
	}
	return partnerID, priority, err == nil, err
}

func (r *Repository) CreateOffer(ctx context.Context, s scope, dispatchID, partnerID string, round, priority int) error {
	_, err := r.db(ctx).Exec(ctx, `
		WITH offer AS (
		  INSERT INTO delivery_dispatch_offers(organization_id,location_id,dispatch_id,partner_id,round,priority)
		  VALUES ($1,$2,$3,$4,$5,$6)
		  RETURNING id
		)
		INSERT INTO delivery_partner_messages(organization_id,location_id,dispatch_id,offer_id,kind)
		SELECT $1,$2,$3,offer.id,'offer' FROM offer`,
		s.OrganizationID, s.LocationID, dispatchID, partnerID, round, priority)
	return err
}

// ExpireOverdueOffers vence las ofertas sin respuesta 3 minutos después de
// enviadas, o las que no se pudieron enviar en 5 minutos.
func (r *Repository) ExpireOverdueOffers(ctx context.Context, s scope, dispatchID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatch_offers f SET status=CASE WHEN p.active AND `+partnerCoverageSQL+` THEN 'expired' ELSE 'withdrawn' END
        FROM delivery_partners p,locations l
		WHERE f.dispatch_id=$1 AND f.organization_id=$2 AND f.status='pending'
          AND p.id=f.partner_id AND l.id=f.location_id AND l.organization_id=f.organization_id
		  AND (NOT p.active OR NOT (`+partnerCoverageSQL+`) OR f.expires_at<now() OR (f.expires_at IS NULL AND f.created_at<now()-make_interval(secs => $3)))`,
		dispatchID, s.OrganizationID, domain.UndeliveredOfferTimeout.Seconds())
	return err
}

func (r *Repository) HasPendingOffer(ctx context.Context, s scope, dispatchID string) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM delivery_dispatch_offers WHERE dispatch_id=$1 AND organization_id=$2 AND status='pending')`,
		dispatchID, s.OrganizationID).Scan(&ok)
	return ok, err
}

func (r *Repository) WithdrawPendingOffers(ctx context.Context, s scope, dispatchID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatch_offers SET status='withdrawn'
		WHERE dispatch_id=$1 AND organization_id=$2 AND status='pending'`, dispatchID, s.OrganizationID)
	return err
}

func (r *Repository) RespondOffer(ctx context.Context, s scope, offerID, status string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatch_offers
		SET status=$3,responded_at=CASE WHEN $3 IN ('accepted','rejected') THEN now() ELSE responded_at END
		WHERE id=$1 AND organization_id=$2`, offerID, s.OrganizationID, status)
	return err
}

// Accept registra la aceptación, retira las demás ofertas pendientes y deja la
// central como responsable del reparto.
func (r *Repository) Accept(ctx context.Context, s scope, dispatchID, offerID string) error {
	if err := r.RespondOffer(ctx, s, offerID, domain.OfferAccepted); err != nil {
		return err
	}
	if _, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatch_offers SET status='withdrawn'
		WHERE dispatch_id=$1 AND organization_id=$2 AND status='pending' AND id<>$3`, dispatchID, s.OrganizationID, offerID); err != nil {
		return err
	}
	// El envío pasa a la central: se recuerda en el despacho (si ya se había
	// cedido en una vuelta anterior, se conserva el monto recordado).
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches d
		SET status='accepted',accepted_partner_id=f.partner_id,accepted_at=now(),updated_at=now(),
		    partner_fee=CASE WHEN d.fee_released THEN d.partner_fee ELSE o.delivery_fee END
		FROM delivery_dispatch_offers f, orders o
		WHERE d.id=$1 AND d.organization_id=$2 AND f.id=$3 AND f.dispatch_id=d.id
		  AND o.id=d.order_id AND o.organization_id=d.organization_id`, dispatchID, s.OrganizationID, offerID)
	return err
}

func (r *Repository) FeeState(ctx context.Context, s scope, orderID string) (application.FeeState, bool, error) {
	var st application.FeeState
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text,status,COALESCE(partner_fee,0)::float8,fee_released
		FROM delivery_dispatches WHERE order_id=$1 AND organization_id=$2 AND location_id=$3`,
		orderID, s.OrganizationID, s.LocationID).Scan(&st.DispatchID, &st.Status, &st.PartnerFee, &st.Released)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, false, nil
	}
	return st, err == nil, err
}

func (r *Repository) MarkFeeReleased(ctx context.Context, s scope, dispatchID string, released bool) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches SET fee_released=$3,updated_at=now()
		WHERE id=$1 AND organization_id=$2`, dispatchID, s.OrganizationID, released)
	return err
}

func (r *Repository) AcceptedOffer(ctx context.Context, s scope, dispatchID string) (string, bool, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT id::text FROM delivery_dispatch_offers
		WHERE dispatch_id=$1 AND organization_id=$2 AND status='accepted'
          AND round=(SELECT round FROM delivery_dispatches WHERE id=$1)
		ORDER BY responded_at DESC NULLS LAST,created_at DESC
		LIMIT 1`, dispatchID, s.OrganizationID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return id, err == nil, err
}

func (r *Repository) EnqueueMessage(ctx context.Context, s scope, dispatchID, offerID, kind string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO delivery_partner_messages(organization_id,location_id,dispatch_id,offer_id,kind)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (offer_id,kind) DO NOTHING`, s.OrganizationID, s.LocationID, dispatchID, offerID, kind)
	return err
}

func (r *Repository) MarkReadyNotified(ctx context.Context, s scope, dispatchID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches SET ready_notified_at=COALESCE(ready_notified_at,now()),updated_at=now()
		WHERE id=$1 AND organization_id=$2`, dispatchID, s.OrganizationID)
	return err
}

func (r *Repository) MarkPickedUp(ctx context.Context, s scope, dispatchID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE delivery_dispatches SET status='picked_up',picked_up_at=COALESCE(picked_up_at,now()),updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND status IN ('accepted','picked_up')`, dispatchID, s.OrganizationID)
	return err
}

func (r *Repository) Dispatch(ctx context.Context, s scope, orderID string) (domain.Dispatch, error) {
	out := domain.Dispatch{Offers: []domain.DispatchOffer{}}
	var id string
	var accepted, picked *time.Time
	err := r.db(ctx).QueryRow(ctx, `
		SELECT d.id::text,d.order_id::text,d.status,d.round,COALESCE(ap.name,current.name,''),d.accepted_at,d.picked_up_at
		FROM delivery_dispatches d
		LEFT JOIN delivery_partners ap ON ap.id=d.accepted_partner_id
		LEFT JOIN LATERAL (
		  SELECT p.name FROM delivery_dispatch_offers f JOIN delivery_partners p ON p.id=f.partner_id
		  WHERE f.dispatch_id=d.id AND f.status='pending' ORDER BY f.created_at DESC LIMIT 1
		) current ON true
		WHERE d.order_id=$1 AND d.organization_id=$2 AND d.location_id=$3`,
		orderID, s.OrganizationID, s.LocationID).Scan(&id, &out.OrderID, &out.Status, &out.Round, &out.PartnerName, &accepted, &picked)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, application.ErrNotFound
	}
	if err != nil {
		return out, err
	}
	out.AcceptedAt, out.PickedUpAt = optionalStamp(accepted), optionalStamp(picked)
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.name,f.round,f.priority,f.status,f.created_at,f.expires_at,f.responded_at
		FROM delivery_dispatch_offers f JOIN delivery_partners p ON p.id=f.partner_id
		WHERE f.dispatch_id=$1 AND f.organization_id=$2
		ORDER BY f.round,f.priority,f.created_at`, id, s.OrganizationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var o domain.DispatchOffer
		var created time.Time
		var expires, responded *time.Time
		if err = rows.Scan(&o.PartnerName, &o.Round, &o.Priority, &o.Status, &created, &expires, &responded); err != nil {
			return out, err
		}
		o.CreatedAt, o.ExpiresAt, o.RespondedAt = stamp(created), optionalStamp(expires), optionalStamp(responded)
		out.Offers = append(out.Offers, o)
	}
	return out, rows.Err()
}

// DueOrders lista los pedidos con trabajo pendiente para la cascada:
//   - delivery confirmado de un local en modo Fudia, aún sin despacho (12 h);
//   - despacho buscando central sin oferta pendiente o con la oferta vencida;
//   - central aceptó y cocina terminó, sin «listo para recoger» enviado;
//   - despacho activo de un pedido cancelado, o resuelto por el local.
func (r *Repository) DueOrders(ctx context.Context, limit int) ([]application.OrderRef, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT DISTINCT organization_id::text,location_id::text,order_id::text FROM (
		  SELECT o.organization_id,o.location_id,o.id AS order_id
		  FROM orders o
		  JOIN organization_delivery_settings ds
		    ON ds.organization_id=o.organization_id AND ds.enabled
		  WHERE o.channel='delivery' AND o.status IN ('confirmado','preparando','listo') AND o.completed_at IS NULL
		    AND o.created_at>now()-interval '12 hours'
		    AND NOT EXISTS (SELECT 1 FROM delivery_dispatches d WHERE d.order_id=o.id)
		  UNION ALL
		  SELECT d.organization_id,d.location_id,d.order_id
		  FROM delivery_dispatches d
		  WHERE d.status='searching' AND (
		    NOT EXISTS (SELECT 1 FROM delivery_dispatch_offers f WHERE f.dispatch_id=d.id AND f.status='pending')
		    OR EXISTS (
		      SELECT 1 FROM delivery_dispatch_offers f JOIN delivery_partners p ON p.id=f.partner_id
              JOIN locations l ON l.id=f.location_id AND l.organization_id=f.organization_id
		      WHERE f.dispatch_id=d.id AND f.status='pending'
		        AND (NOT p.active OR NOT (`+partnerCoverageSQL+`) OR f.expires_at<now() OR (f.expires_at IS NULL AND f.created_at<now()-make_interval(secs => $2)))))
		  UNION ALL
		  SELECT d.organization_id,d.location_id,d.order_id
		  FROM delivery_dispatches d
		  JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		  WHERE d.status IN ('searching','accepted','unassigned') AND (
		    o.status='cancelado'
		    OR (d.status='searching' AND o.paid_amount>0)
		    OR (d.status='accepted' AND d.ready_notified_at IS NULL AND o.status='listo')
		    OR (d.status<>'accepted' AND (o.completed_at IS NOT NULL OR o.status IN ('en_camino','entregado'))))
		  UNION ALL
		  -- Envío por mover: la central lo tiene y aún no salió del pedido, o el
		  -- local volvió a su delivery propio y debe regresar al pedido.
		  SELECT d.organization_id,d.location_id,d.order_id
		  FROM delivery_dispatches d
		  JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		  WHERE o.completed_at IS NULL AND o.status<>'cancelado' AND (
		    (d.status IN ('accepted','picked_up') AND NOT d.fee_released AND COALESCE(d.partner_fee,0)>0)
		    OR (d.status NOT IN ('accepted','picked_up') AND d.fee_released))
		) due
		LIMIT $1`, limit, domain.UndeliveredOfferTimeout.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.OrderRef{}
	for rows.Next() {
		var ref application.OrderRef
		if err := rows.Scan(&ref.OrganizationID, &ref.LocationID, &ref.OrderID); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

// ClaimMessages reserva por 60 s los mensajes pendientes de las últimas 2 horas
// (hasta 5 intentos) que siguen vigentes: la oferta aún espera respuesta, el
// «listo» es de la central que aceptó, o el aviso de cancelación.
func (r *Repository) ClaimMessages(ctx context.Context, limit int) ([]domain.PartnerMessage, error) {
	rows, err := r.db(ctx).Query(ctx, `
		WITH picked AS (
		  SELECT m.id FROM delivery_partner_messages m
		  JOIN delivery_dispatch_offers f ON f.id=m.offer_id
		  JOIN delivery_dispatches d ON d.id=m.dispatch_id
		  JOIN delivery_partners p ON p.id=f.partner_id
          JOIN locations l ON l.id=f.location_id AND l.organization_id=f.organization_id
          JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		  WHERE m.status='pending' AND m.attempts<5 AND m.created_at>now()-interval '2 hours'
		    AND (m.claimed_until IS NULL OR m.claimed_until<now())
		    AND CASE m.kind
		      WHEN 'offer' THEN f.status='pending' AND f.round=d.round AND d.status='searching' AND p.active AND `+partnerCoverageSQL+` AND o.paid_amount=0 AND o.completed_at IS NULL AND o.status IN ('confirmado','preparando','listo')
		      WHEN 'ready' THEN f.status='accepted' AND f.round=d.round AND d.accepted_partner_id=f.partner_id AND d.status='accepted' AND o.status='listo'
		      ELSE true END
		  ORDER BY m.created_at,m.id
		  LIMIT $1
		  FOR UPDATE OF m SKIP LOCKED
		), claimed AS (
		  UPDATE delivery_partner_messages m SET claimed_until=now()+interval '60 seconds',attempts=m.attempts+1
		  FROM picked WHERE m.id=picked.id
		  RETURNING m.*
		)
		SELECT c.id::text,c.kind,p.name,p.whatsapp_phone,f.action_token::text,
		       o.code,COALESCE(NULLIF(btrim(org.trade_name),''),org.legal_name),l.name,l.address,
		       o.customer_name,o.customer_phone,o.address,o.reference,
		       o.delivery_latitude::float8,o.delivery_longitude::float8,o.delivery_distance_km::float8,o.notes,
		       COALESCE((
		         SELECT json_agg(json_build_object('quantity',i.qty::float8,'name',i.name,'note',i.note) ORDER BY i.created_at,i.id)
		         FROM order_items i WHERE i.order_id=o.id AND i.organization_id=o.organization_id
		       ),'[]'::json)::text,
		       COALESCE(fp.currency_symbol,''),o.subtotal::text,
		       COALESCE(d.partner_fee,o.delivery_fee)::text,(o.subtotal+COALESCE(d.partner_fee,o.delivery_fee))::text,c.created_at,COALESCE(NULLIF(c.routing_key,''),CASE WHEN c.legacy_route THEN legacy.routing_key END,'')
		FROM claimed c
		JOIN delivery_dispatch_offers f ON f.id=c.offer_id
		JOIN delivery_partners p ON p.id=f.partner_id
		JOIN delivery_dispatches d ON d.id=c.dispatch_id
		JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		JOIN organizations org ON org.id=c.organization_id
		JOIN locations l ON l.id=c.location_id AND l.organization_id=c.organization_id
		LEFT JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
 LEFT JOIN LATERAL (SELECT r.routing_key FROM platform_whatsapp_routes r WHERE r.country_code=fp.country_code AND r.legacy ORDER BY r.created_at,r.id LIMIT 1) legacy ON true
		ORDER BY c.created_at,c.id`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PartnerMessage{}
	for rows.Next() {
		var m domain.PartnerMessage
		var items string
		var created time.Time
		if err := rows.Scan(&m.ID, &m.Kind, &m.PartnerName, &m.PartnerPhone, &m.ActionToken,
			&m.OrderCode, &m.OrganizationName, &m.LocationName, &m.LocationAddress,
			&m.CustomerName, &m.CustomerPhone, &m.Address, &m.Reference,
			&m.Latitude, &m.Longitude, &m.DistanceKm, &m.Notes, &items,
			&m.CurrencySymbol, &m.Subtotal, &m.DeliveryFee, &m.Total, &created, &m.RoutingKey); err != nil {
			return nil, err
		}
		m.Items = []domain.MessageItem{}
		if err := json.Unmarshal([]byte(items), &m.Items); err != nil {
			return nil, err
		}
		if m.Kind == domain.MessageOffer {
			m.ExpiresInSeconds = int(domain.OfferTimeout.Seconds())
		}
		m.CreatedAt = stamp(created)
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkMessageDispatched confirma el envío; para una oferta inicia sus 3 minutos.
func (r *Repository) MarkMessageDispatched(ctx context.Context, id string) error {
	var exists bool
	if err := r.db(ctx).QueryRow(ctx, `
		WITH done AS (
		  UPDATE delivery_partner_messages SET status='dispatched',dispatched_at=now(),claimed_until=NULL
		  WHERE id=$1 AND status='pending' RETURNING offer_id,kind
		), timer AS (
		  UPDATE delivery_dispatch_offers f SET expires_at=now()+make_interval(secs => $2)
		  FROM done WHERE done.kind='offer' AND f.id=done.offer_id AND f.status='pending' AND f.expires_at IS NULL
		  RETURNING f.id
		)
		SELECT EXISTS (SELECT 1 FROM done) OR EXISTS (SELECT 1 FROM delivery_partner_messages WHERE id=$1)`,
		id, domain.OfferTimeout.Seconds()).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return application.ErrNotFound
	}
	return nil
}

// LockOfferByToken carga la oferta del botón y bloquea su despacho.
func (r *Repository) LockOfferByToken(ctx context.Context, token string) (application.OfferState, error) {
	var st application.OfferState
	// All operations lock order -> dispatch -> partner. Payments also lock the
	// order, so acceptance and pickup cannot race collection or UseOwn.
	var orderID, orgID, locationID string
	err := r.db(ctx).QueryRow(ctx, `SELECT o.id::text,o.organization_id::text,o.location_id::text
      FROM delivery_dispatch_offers f JOIN delivery_dispatches d ON d.id=f.dispatch_id
      JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
      WHERE f.action_token=$1::uuid FOR UPDATE OF o`, token).Scan(&orderID, &orgID, &locationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, application.ErrNotFound
	}
	if err != nil {
		return st, err
	}
	err = r.db(ctx).QueryRow(ctx, `
		SELECT f.organization_id::text,f.location_id::text,f.id::text,d.id::text,d.order_id::text,
		       f.partner_id::text,p.whatsapp_phone,f.status,d.status,o.status,
		       (f.expires_at IS NOT NULL AND f.expires_at<now()),
		       o.code,COALESCE(NULLIF(btrim(org.trade_name),''),org.legal_name),l.name,
               f.round=d.round,COALESCE(d.accepted_partner_id=f.partner_id,false),
               p.active AND `+partnerCoverageSQL+`,o.paid_amount>0,o.completed_at IS NOT NULL
		FROM delivery_dispatch_offers f
		JOIN delivery_dispatches d ON d.id=f.dispatch_id
		JOIN delivery_partners p ON p.id=f.partner_id
		JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		JOIN organizations org ON org.id=d.organization_id
		JOIN locations l ON l.id=d.location_id AND l.organization_id=d.organization_id
		WHERE f.action_token=$1::uuid
		FOR UPDATE OF d FOR SHARE OF p`, token).Scan(
		&st.OrganizationID, &st.LocationID, &st.OfferID, &st.DispatchID, &st.OrderID,
		&st.PartnerID, &st.PartnerPhone, &st.OfferStatus, &st.DispatchStatus, &st.OrderStatus,
		&st.Expired, &st.OrderCode, &st.OrganizationName, &st.LocName, &st.CurrentRound, &st.Assigned, &st.Eligible, &st.HasPayments, &st.Completed)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, application.ErrNotFound
	}
	return st, err
}
