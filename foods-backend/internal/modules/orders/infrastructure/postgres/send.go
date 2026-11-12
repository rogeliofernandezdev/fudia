package postgres

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

// HoldRoundItems deja sin enviar (status «nuevo», fuera de Cocina) los productos
// de una ronda pedida por Fudia Concierge sobre una cuenta ya enviada.
func (r *Orders) HoldRoundItems(ctx context.Context, s scope, orderID, roundID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE order_service_items si SET status='nuevo',updated_at=now()
		FROM order_items i
		WHERE i.id=si.order_item_id AND i.organization_id=si.organization_id
		  AND si.order_id=$1 AND si.organization_id=$2 AND i.kitchen_round_id=$3::uuid`, orderID, s.OrganizationID, roundID)
	return tag.RowsAffected(), orderError(err)
}

// HasUnsentItems indica si el pedido tiene productos guardados sin enviar a Cocina.
func (r *Orders) HasUnsentItems(ctx context.Context, s scope, orderID string) (bool, error) {
	var pending bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2 AND status='nuevo')`, orderID, s.OrganizationID).Scan(&pending)
	return pending, orderError(err)
}

// UnsentRecipients devuelve los WhatsApp que pidieron lo que está por enviarse:
// cada ronda de mesa recuerda su conversación; un pedido sin mesa avisa a su cliente.
func (r *Orders) UnsentRecipients(ctx context.Context, s scope, orderID string) ([]domain.SendRecipient, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT DISTINCT q.conversation_id,q.customer_phone,q.round_id::text
		FROM concierge_order_requests q
		WHERE q.organization_id=$1 AND q.location_id=$2 AND q.order_id=$3
		  AND q.round_id IS NOT NULL
		  AND EXISTS(
		    SELECT 1 FROM order_items i
		    JOIN order_service_items si ON si.order_item_id=i.id AND si.organization_id=i.organization_id
		    WHERE i.kitchen_round_id=q.round_id AND i.organization_id=q.organization_id AND si.status='nuevo')
		UNION
		SELECT d.conversation_id,o.customer_phone,o.id::text
		FROM concierge_delivery_requests d
		JOIN orders o ON o.id=d.order_id AND o.organization_id=d.organization_id
		WHERE d.organization_id=$1 AND d.location_id=$2 AND d.order_id=$3
		  AND o.status='nuevo'`, s.OrganizationID, s.LocationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SendRecipient{}
	for rows.Next() {
		var item domain.SendRecipient
		if err := rows.Scan(&item.ConversationID, &item.Phone, &item.Key); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// SendUnsentItems pasa a Cocina/Barra lo guardado sin enviar. La hora del ticket
// empieza al enviarlo, no cuando el cliente lo pidió.
func (r *Orders) SendUnsentItems(ctx context.Context, s scope, orderID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE order_service_items
		SET status=CASE WHEN destination='direct' THEN 'listo' ELSE 'confirmado' END,created_at=now(),updated_at=now()
		WHERE order_id=$1 AND organization_id=$2 AND status='nuevo'`, orderID, s.OrganizationID)
	return tag.RowsAffected(), orderError(err)
}

// AssignAttendant registra a quien envió la comanda como responsable si aún no hay uno.
func (r *Orders) AssignAttendant(ctx context.Context, s scope, orderID, userID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE orders SET waiter_id=(SELECT id FROM users WHERE id=$4::uuid AND organization_id=$2),updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3 AND waiter_id IS NULL`, orderID, s.OrganizationID, s.LocationID, userID)
	return err
}
