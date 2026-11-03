package postgres

import (
	"context"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
)

func (r *Notices) InsertCustomer(ctx context.Context, s scope, in domain.CustomerInput) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO customer_notices(organization_id,location_id,order_id,kind,customer_phone,conversation_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (order_id,kind) DO NOTHING`,
		s.OrganizationID, s.LocationID, in.OrderID, in.Kind, in.CustomerPhone, in.ConversationID)
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
		       COALESCE(NULLIF(btrim(org.trade_name),''),org.legal_name),l.name,c.created_at
		FROM claimed c
		JOIN orders o ON o.id=c.order_id AND o.organization_id=c.organization_id
		JOIN organizations org ON org.id=c.organization_id
		JOIN locations l ON l.id=c.location_id AND l.organization_id=c.organization_id
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
			&item.OrganizationName, &item.LocationName, &created); err != nil {
			return nil, err
		}
		item.Fulfillment = domain.FulfillmentFor(channel)
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
