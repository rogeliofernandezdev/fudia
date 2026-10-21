package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (a *API) requirePreparationPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		permission := "kitchen.manage"
		if strings.HasSuffix(r.PathValue("id"), "~bar") {
			permission = "bar.manage"
		}
		a.requirePermission(permission, next).ServeHTTP(w, r)
	})
}

func (a *API) serviceKitchenTickets(r *http.Request, channel, destination string) ([]kitchenTicket, error) {
	s := r.Context().Value(scopeKey{}).(scope)
	rows, err := a.db.Query(r.Context(), `SELECT COALESCE(i.kitchen_round_id,o.id)::text,COALESCE(kr.sequence,0),
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
 AND ($3='' OR o.channel=$3) AND ($4='' OR si.destination=$4)
 ORDER BY si.created_at,si.id`, s.OrganizationID, s.LocationID, channel, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tickets := []kitchenTicket{}
	indices := map[string]int{}
	for rows.Next() {
		var t kitchenTicket
		var item orderItem
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
			t.Items = []orderItem{}
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

// Missing kitchen snapshots may need historical lookup; the fallback must
// verify the whole order has no service snapshots, under its order lock.
func (a *API) updateServiceKitchenTicket(w http.ResponseWriter, r *http.Request, status string) bool {
	s := r.Context().Value(scopeKey{}).(scope)
	id := r.PathValue("id")
	destination := "kitchen"
	if strings.HasSuffix(id, "~bar") {
		destination = "bar"
		id = strings.TrimSuffix(id, "~bar")
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos actualizar la comanda.")
		return true
	}
	defer tx.Rollback(r.Context())
	var orderID string
	err = tx.QueryRow(r.Context(), `SELECT o.id::text FROM orders o WHERE o.organization_id=$2 AND o.location_id=$3 AND o.completed_at IS NULL AND o.status<>'cancelado'
 AND EXISTS(SELECT 1 FROM order_items i JOIN order_service_items si ON si.order_item_id=i.id AND si.organization_id=i.organization_id WHERE i.order_id=o.id AND i.organization_id=o.organization_id AND COALESCE(i.kitchen_round_id,o.id)=$1::uuid AND si.destination=$4)
 FOR UPDATE`, id, s.OrganizationID, s.LocationID, destination).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		if destination == "bar" {
			fail(w, 404, "kitchen_ticket_not_found", "La comanda no existe.")
			return true
		}
		return false
	}
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos validar la comanda.")
		return true
	}
	var current string
	err = tx.QueryRow(r.Context(), `SELECT CASE WHEN bool_or(si.status='preparando') THEN 'preparando' WHEN bool_or(si.status='confirmado') THEN 'confirmado' ELSE 'listo' END
 FROM order_service_items si JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id
 WHERE si.order_id=$1 AND si.organization_id=$2 AND COALESCE(i.kitchen_round_id,i.order_id)=$3::uuid AND si.destination=$4 AND si.status<>'entregado'`, orderID, s.OrganizationID, id, destination).Scan(&current)
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos consultar la preparación.")
		return true
	}
	if !validKitchenTransition(current, status) {
		fail(w, 409, "invalid_kitchen_transition", "La comanda cambió de estado. Actualiza antes de continuar.")
		return true
	}
	_, err = tx.Exec(r.Context(), `UPDATE order_service_items si SET status=$5,updated_at=now() FROM order_items i
 WHERE si.order_item_id=i.id AND si.organization_id=i.organization_id AND si.order_id=$1 AND si.organization_id=$2 AND COALESCE(i.kitchen_round_id,i.order_id)=$3::uuid AND si.destination=$4 AND si.status=$6`, orderID, s.OrganizationID, id, destination, status, current)
	if err == nil {
		_, err = tx.Exec(r.Context(), `UPDATE order_kitchen_rounds SET status=CASE WHEN EXISTS(SELECT 1 FROM order_service_items si JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id WHERE i.kitchen_round_id=$1 AND i.organization_id=$2 AND si.status='preparando') THEN 'preparando' WHEN EXISTS(SELECT 1 FROM order_service_items si JOIN order_items i ON i.id=si.order_item_id AND i.organization_id=si.organization_id WHERE i.kitchen_round_id=$1 AND i.organization_id=$2 AND si.status='confirmado') THEN 'confirmado' ELSE 'listo' END,updated_at=now() WHERE id=$1 AND organization_id=$2`, id, s.OrganizationID)
	}
	if err == nil {
		err = syncOrderServiceStatus(r.Context(), tx, s, orderID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos guardar la preparación.")
		return true
	}
	a.audit(r, "preparation."+status, "order", orderID)
	w.WriteHeader(204)
	return true
}
