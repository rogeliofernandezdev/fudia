package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/application"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

func (r *Orders) LegacyKitchenOrders(ctx context.Context, s scope, channel, destination string) ([]domain.Order, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT `+orderColumns+`
		FROM orders
		WHERE organization_id=$1
		  AND location_id=$2
		  AND status IN ('confirmado','preparando','listo')
		  AND ($3='' OR channel=$3 OR ($3 IN ('recojo','mostrador') AND channel IN ('recojo','mostrador')) OR ($3='salon' AND channel='whatsapp'))
 AND $4 <> 'bar'
		  AND NOT EXISTS(SELECT 1 FROM order_service_items si WHERE si.order_id=orders.id AND si.organization_id=orders.organization_id)
 AND NOT EXISTS (
		    SELECT 1 FROM order_kitchen_rounds kr
		    WHERE kr.order_id=orders.id
		      AND kr.organization_id=orders.organization_id
		      AND kr.location_id=orders.location_id
		  )
		ORDER BY
		  CASE status WHEN 'preparando' THEN 0 WHEN 'confirmado' THEN 1 ELSE 2 END,
		  updated_at ASC,created_at ASC
		LIMIT 150`, s.OrganizationID, s.LocationID, channel, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Order{}
	for rows.Next() {
		v, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Orders) KitchenRounds(ctx context.Context, s scope, channel, destination string) ([]domain.KitchenTicket, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT kr.id::text,kr.sequence,kr.status,
		       o.id::text,o.code,o.channel,o.customer_name,
		       COALESCE((SELECT t.name FROM tables t WHERE t.id=o.table_id),''),
		       o.notes,
		       to_char(kr.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       to_char(kr.updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM order_kitchen_rounds kr
		JOIN orders o ON o.id=kr.order_id AND o.organization_id=kr.organization_id
		WHERE kr.organization_id=$1 AND kr.location_id=$2
		  AND kr.status IN ('confirmado','preparando','listo')
		  AND o.status NOT IN ('entregado','cancelado')
 AND NOT EXISTS(SELECT 1 FROM order_service_items si WHERE si.order_id=o.id AND si.organization_id=o.organization_id)
		  AND ($3='' OR o.channel=$3 OR ($3 IN ('recojo','mostrador') AND o.channel IN ('recojo','mostrador')) OR ($3='salon' AND o.channel='whatsapp'))
 AND $4 <> 'bar'
		ORDER BY
		  CASE kr.status WHEN 'preparando' THEN 0 WHEN 'confirmado' THEN 1 ELSE 2 END,
		  kr.updated_at ASC,kr.created_at ASC
		LIMIT 150
	`, s.OrganizationID, s.LocationID, channel, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.KitchenTicket{}
	for rows.Next() {
		var v domain.KitchenTicket
		if err = rows.Scan(&v.ID, &v.RoundNumber, &v.Status, &v.OrderID, &v.Code, &v.Channel, &v.CustomerName, &v.TableName, &v.Notes, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Orders) ServiceTickets(ctx context.Context, s scope, channel, destination string) ([]domain.KitchenTicket, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT COALESCE(i.kitchen_round_id,o.id)::text,COALESCE(kr.sequence,0),
 si.destination,si.status,o.id::text,o.code,o.channel,o.customer_name,COALESCE(t.name,''),o.notes,
 to_char(si.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),to_char(si.updated_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),
 si.id::text,COALESCE(si.product_id::text,''),si.name,si.qty::text,
 i.note||COALESCE((SELECT ' · '||string_agg(m.option_name,' · ' ORDER BY m.created_at,m.id) FROM order_item_modifier_selections m WHERE m.order_item_id=i.id AND m.organization_id=i.organization_id),''),COALESCE(p.prep_minutes,0)
 FROM order_service_items si
 JOIN orders o ON o.id=si.order_id AND o.organization_id=si.organization_id
 JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id
 LEFT JOIN order_kitchen_rounds kr ON kr.id=i.kitchen_round_id AND kr.organization_id=i.organization_id
 LEFT JOIN products p ON p.id=si.product_id AND p.organization_id=si.organization_id
 LEFT JOIN tables t ON t.id=o.table_id AND t.organization_id=o.organization_id AND t.location_id=o.location_id
 WHERE o.organization_id=$1 AND o.location_id=$2 AND o.completed_at IS NULL AND o.status<>'cancelado'
 AND si.destination IN ('kitchen','bar') AND si.status IN ('confirmado','preparando','listo')
 AND ($3='' OR o.channel=$3 OR ($3 IN ('recojo','mostrador') AND o.channel IN ('recojo','mostrador')) OR ($3='salon' AND o.channel='whatsapp')) AND ($4='' OR si.destination=$4)
 ORDER BY si.created_at,si.id`, s.OrganizationID, s.LocationID, channel, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tickets := []domain.KitchenTicket{}
	indices := map[string]int{}
	for rows.Next() {
		var t domain.KitchenTicket
		var item domain.OrderItem
		var itemStatus string
		var minutes int
		if err = rows.Scan(&t.ID, &t.RoundNumber, &t.Destination, &itemStatus, &t.OrderID, &t.Code, &t.Channel, &t.CustomerName, &t.TableName, &t.Notes, &t.CreatedAt, &t.UpdatedAt, &item.ID, &item.ProductID, &item.Name, &item.Qty, &item.Note, &minutes); err != nil {
			return nil, err
		}
		if t.Destination == "bar" {
			t.ID += "~bar"
		}
		item.UnitPrice = "0.00"
		item.ItemType = "product"
		t.Status = itemStatus
		index, ok := indices[t.ID]
		if !ok {
			index = len(tickets)
			indices[t.ID] = index
			t.Items = []domain.OrderItem{}
			tickets = append(tickets, t)
		}
		current := &tickets[index]
		current.Items = append(current.Items, item)
		if itemStatus == "preparando" || (itemStatus == "confirmado" && current.Status == "listo") {
			current.Status = itemStatus
		}
		if minutes > 0 && (current.TargetMinutes == nil || minutes > *current.TargetMinutes) {
			value := minutes
			current.TargetMinutes = &value
		}
		if t.UpdatedAt > current.UpdatedAt {
			current.UpdatedAt = t.UpdatedAt
		}
	}
	return tickets, rows.Err()
}
func (r *Orders) TargetMinutes(ctx context.Context, s scope, orderID, roundID string) (*int, error) {
	var target int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(MAX(prep_minutes),0)
		FROM (
			SELECT p.prep_minutes
			FROM order_items oi
			JOIN products p
			  ON p.id=oi.product_id AND p.organization_id=oi.organization_id
			WHERE oi.order_id=$1 AND oi.organization_id=$2
			  AND ($3='' OR oi.kitchen_round_id::text=$3)
			  AND p.prep_minutes IS NOT NULL AND p.prep_minutes>0
			UNION ALL
			SELECT p.prep_minutes
			FROM order_item_combo_selections s
			JOIN order_items oi
			  ON oi.id=s.order_item_id AND oi.organization_id=s.organization_id
			JOIN products p
			  ON p.id=s.option_product_id AND p.organization_id=s.organization_id
			WHERE oi.order_id=$1 AND oi.organization_id=$2
			  AND ($3='' OR oi.kitchen_round_id::text=$3)
			  AND p.prep_minutes IS NOT NULL AND p.prep_minutes>0
		) preparation_times`, orderID, s.OrganizationID, roundID).Scan(&target)
	if err != nil {
		return nil, err
	}
	if target <= 0 {
		return nil, nil
	}
	return &target, nil
}
func (r *Orders) LockStationOrder(ctx context.Context, s scope, id string, destination string) (string, error) {
	var orderID string
	err := r.db(ctx).QueryRow(ctx, `SELECT o.id::text FROM orders o WHERE o.organization_id=$2 AND o.location_id=$3 AND o.completed_at IS NULL AND o.status<>'cancelado'
 AND EXISTS(SELECT 1 FROM order_items i JOIN order_service_items si ON si.order_item_id=i.id AND si.organization_id=i.organization_id WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND COALESCE(i.kitchen_round_id,o.id)=$1::uuid AND si.destination=$4)
 FOR UPDATE`, id, s.OrganizationID, s.LocationID, destination).Scan(&orderID)
	return orderID, orderError(err)
}
func (r *Orders) StationStatus(ctx context.Context, s scope, orderID string, id string, destination string) (string, error) {
	var current string
	err := r.db(ctx).QueryRow(ctx, `SELECT CASE WHEN bool_or(si.status='preparando') THEN 'preparando' WHEN bool_or(si.status='confirmado') THEN 'confirmado' ELSE 'listo' END
 FROM order_service_items si JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id
 WHERE si.order_id=$1 AND si.organization_id=$2 AND COALESCE(i.kitchen_round_id,i.order_id)=$3::uuid AND si.destination=$4 AND si.status<>'entregado'`, orderID, s.OrganizationID, id, destination).Scan(&current)
	return current, orderError(err)
}
func (r *Orders) UpdateStationItems(ctx context.Context, s scope, orderID string, id string, destination string, status string, current string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_service_items si SET status=$5,updated_at=now() FROM order_items i
 WHERE si.order_item_id=i.id AND si.organization_id=i.organization_id AND si.order_id=$1 AND si.organization_id=$2 AND COALESCE(i.kitchen_round_id,i.order_id)=$3::uuid AND si.destination=$4 AND si.status=$6`, orderID, s.OrganizationID, id, destination, status, current)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) UpdateStationRound(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE order_kitchen_rounds SET status=CASE WHEN EXISTS(SELECT 1 FROM order_service_items si JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id WHERE i.kitchen_round_id=$1 AND i.organization_id=$2 AND si.status='preparando') THEN 'preparando' WHEN EXISTS(SELECT 1 FROM order_service_items si JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id WHERE i.kitchen_round_id=$1 AND i.organization_id=$2 AND si.status='confirmado') THEN 'confirmado' ELSE 'listo' END,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, s.OrganizationID)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) LockHistoricalRound(ctx context.Context, s scope, id string) (string, string, error) {
	var roundCurrent string
	var roundOrderID string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT kr.status,kr.order_id::text
		FROM order_kitchen_rounds kr
		JOIN orders o ON o.id=kr.order_id AND o.organization_id=kr.organization_id
		WHERE kr.id=$1 AND kr.organization_id=$2 AND kr.location_id=$3
		  AND o.status NOT IN ('entregado','cancelado')
		  AND o.completed_at IS NULL
		FOR UPDATE OF kr,o
	`, id, s.OrganizationID, s.LocationID).Scan(&roundCurrent, &roundOrderID)
	return roundCurrent, roundOrderID, orderError(err)
}
func (r *Orders) UpdateHistoricalRound(ctx context.Context, s scope, id string, in application.OrderStatusInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			UPDATE order_kitchen_rounds
			SET status=$4,updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
		`, id, s.OrganizationID, s.LocationID, in.Status)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) LockHistoricalOrder(ctx context.Context, s scope, id string) (string, error) {
	var current string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT status
		FROM orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		  AND completed_at IS NULL
		FOR UPDATE
	`, id, s.OrganizationID, s.LocationID).Scan(&current)
	return current, orderError(err)
}
func (r *Orders) UpdateHistoricalOrder(ctx context.Context, s scope, id string, in application.OrderStatusInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE orders
		SET status=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, id, s.OrganizationID, s.LocationID, in.Status)
	return tag.RowsAffected(), orderError(err)
}
func (r *Orders) KitchenRoundCounts(ctx context.Context, s scope, orderID string) (int, int, int, error) {
	var confirmed int
	var preparing int
	var ready int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE status='confirmado'),
		  count(*) FILTER (WHERE status='preparando'),
		  count(*) FILTER (WHERE status='listo')
		FROM order_kitchen_rounds
		WHERE order_id=$1 AND organization_id=$2 AND location_id=$3
	`, orderID, s.OrganizationID, s.LocationID).Scan(&confirmed, &preparing, &ready)
	return confirmed, preparing, ready, orderError(err)
}
func (r *Orders) SyncHistoricalKitchenStatus(ctx context.Context, s scope, orderID string, next string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE orders
		SET status=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		  AND status NOT IN ('entregado','cancelado')
	`, orderID, s.OrganizationID, s.LocationID, next)
	return tag.RowsAffected(), orderError(err)
}
