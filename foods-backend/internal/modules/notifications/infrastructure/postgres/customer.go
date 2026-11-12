package postgres

import (
	"context"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
)

func (r *Notices) InsertCustomer(ctx context.Context, s scope, in domain.CustomerInput) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO customer_notices(organization_id,location_id,order_id,kind,customer_phone,conversation_id,dedupe_key)
		SELECT $1,$2,$3,$4,$5,$6,$7
        WHERE $5<>'' OR EXISTS (SELECT 1 FROM concierge_conversation_origins c
          JOIN platform_whatsapp_routes r ON r.routing_key=c.routing_key AND r.provider='FUDIA_WEB'
          WHERE c.conversation_id=$6 AND c.organization_id=$1 AND c.location_id=$2)
		ON CONFLICT (order_id,kind,dedupe_key) DO NOTHING`,
		s.OrganizationID, s.LocationID, in.OrderID, in.Kind, in.CustomerPhone, in.ConversationID, in.DedupeKey)
	return err
}

// ClaimCustomer reserva por 60 s los avisos pendientes de las últimas 2 horas
// (como máximo 5 intentos) de pedidos no cancelados. SKIP LOCKED permite varios
// workers de Concierge sin entregar el mismo aviso dos veces a la vez.
func (r *Notices) ClaimCustomer(ctx context.Context, limit int) ([]domain.CustomerNotice, error) {
	rows, err := r.db(ctx).Query(ctx, `
		WITH picked AS (
		  SELECT n.id FROM customer_notices n
		  JOIN orders o ON o.id=n.order_id AND o.organization_id=n.organization_id
		  WHERE n.status='pending' AND n.attempts<5 AND n.created_at>now()-interval '2 hours'
		    AND (n.claimed_until IS NULL OR n.claimed_until<now()) AND o.status<>'cancelado'
		  ORDER BY n.created_at,n.id
		  LIMIT $1
		  FOR UPDATE OF n SKIP LOCKED
		), claimed AS (
		  UPDATE customer_notices n SET claimed_until=now()+interval '60 seconds',attempts=n.attempts+1
		  FROM picked WHERE n.id=picked.id
		  RETURNING n.*
		)
		SELECT c.id::text,c.kind,c.order_id::text,o.code,o.channel,c.customer_phone,c.conversation_id,
		       COALESCE(NULLIF(btrim(org.trade_name),''),org.legal_name),l.name,c.created_at,
 COALESCE(NULLIF(c.routing_key,''),origin.routing_key,legacy.routing_key,''),COALESCE(NULLIF(c.recipient,''),origin.recipient,c.customer_phone)
		FROM claimed c
		JOIN orders o ON o.id=c.order_id AND o.organization_id=c.organization_id
		JOIN organizations org ON org.id=c.organization_id
		JOIN locations l ON l.id=c.location_id AND l.organization_id=c.organization_id
 LEFT JOIN concierge_conversation_origins origin ON origin.conversation_id=c.conversation_id AND origin.organization_id=c.organization_id AND origin.location_id=c.location_id
 LEFT JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
 LEFT JOIN LATERAL (SELECT r.routing_key FROM platform_whatsapp_routes r WHERE r.country_code=fp.country_code AND r.legacy ORDER BY r.created_at,r.id LIMIT 1) legacy ON true
		ORDER BY c.created_at,c.id`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CustomerNotice{}
	for rows.Next() {
		var item domain.CustomerNotice
		var channel string
		var created time.Time
		if err := rows.Scan(&item.ID, &item.Kind, &item.OrderID, &item.OrderCode, &channel, &item.CustomerPhone, &item.ConversationID,
			&item.OrganizationName, &item.LocationName, &created, &item.RoutingKey, &item.Recipient); err != nil {
			return nil, err
		}
		item.Fulfillment = domain.FulfillmentFor(channel)
		if item.Kind == domain.CustomerOrderSent || item.Kind == domain.CustomerOrderOnTheWay {
			item.Fulfillment = domain.ChannelFulfillment(channel)
		}
		item.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Notices) MarkCustomerDispatched(ctx context.Context, id string) error {
	var exists bool
	if err := r.db(ctx).QueryRow(ctx, `
		WITH done AS (
		  UPDATE customer_notices SET status='dispatched',dispatched_at=now(),claimed_until=NULL
		  WHERE id=$1 AND status='pending' RETURNING id
		)
		SELECT EXISTS (SELECT 1 FROM done) OR EXISTS (SELECT 1 FROM customer_notices WHERE id=$1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return application.ErrNotFound
	}
	return nil
}
