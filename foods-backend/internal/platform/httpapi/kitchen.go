package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type kitchenTicket struct {
	ID            string      `json:"id"`
	Code          string      `json:"code"`
	Channel       string      `json:"channel"`
	Status        string      `json:"status"`
	CustomerName  string      `json:"customerName"`
	TableName     string      `json:"tableName"`
	Notes         string      `json:"notes"`
	CreatedAt     string      `json:"createdAt"`
	UpdatedAt     string      `json:"updatedAt"`
	TargetMinutes *int        `json:"targetMinutes"`
	Items         []orderItem `json:"items"`
}

func validKitchenTransition(current, next string) bool {
	return (current == "confirmado" && next == "preparando") ||
		(current == "preparando" && next == "listo")
}

func (a *API) kitchenTargetMinutes(r *http.Request, orderID, organizationID string) (*int, error) {
	var target int
	err := a.db.QueryRow(r.Context(), `
		SELECT COALESCE(MAX(prep_minutes),0)
		FROM (
			SELECT p.prep_minutes
			FROM order_items oi
			JOIN products p
			  ON p.id=oi.product_id AND p.organization_id=oi.organization_id
			WHERE oi.order_id=$1 AND oi.organization_id=$2
			  AND p.prep_minutes IS NOT NULL AND p.prep_minutes>0
			UNION ALL
			SELECT p.prep_minutes
			FROM order_item_combo_selections s
			JOIN order_items oi
			  ON oi.id=s.order_item_id AND oi.organization_id=s.organization_id
			JOIN products p
			  ON p.id=s.option_product_id AND p.organization_id=s.organization_id
			WHERE oi.order_id=$1 AND oi.organization_id=$2
			  AND p.prep_minutes IS NOT NULL AND p.prep_minutes>0
		) preparation_times`, orderID, organizationID).Scan(&target)
	if err != nil {
		return nil, err
	}
	if target <= 0 {
		return nil, nil
	}
	return &target, nil
}

func (a *API) listKitchenTickets(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	if channel != "" && !validOrderChannel(channel) {
		fail(w, 400, "invalid_channel", "El canal seleccionado no es válido.")
		return
	}

	rows, err := a.db.Query(r.Context(), `
		SELECT `+orderColumns+`
		FROM orders
		WHERE organization_id=$1
		  AND location_id=$2
		  AND status IN ('confirmado','preparando','listo')
		  AND ($3='' OR channel=$3)
		ORDER BY
		  CASE status WHEN 'preparando' THEN 0 WHEN 'confirmado' THEN 1 ELSE 2 END,
		  updated_at ASC,created_at ASC
		LIMIT 150`, s.OrganizationID, s.LocationID, channel)
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
		return
	}

	orders := []order{}
	for rows.Next() {
		item, scanErr := scanOrder(rows)
		if scanErr != nil {
			rows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
			return
		}
		orders = append(orders, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		fail(w, 503, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
		return
	}
	rows.Close()

	tickets := make([]kitchenTicket, 0, len(orders))
	counts := map[string]int{"confirmado": 0, "preparando": 0, "listo": 0}
	for _, item := range orders {
		items, itemsErr := loadOrderItems(r.Context(), a.db, item.ID, s.OrganizationID)
		if itemsErr != nil {
			fail(w, 503, "kitchen_unavailable", "No pudimos cargar los productos de las comandas.")
			return
		}
		target, targetErr := a.kitchenTargetMinutes(r, item.ID, s.OrganizationID)
		if targetErr != nil {
			fail(w, 503, "kitchen_unavailable", "No pudimos calcular los tiempos de preparación.")
			return
		}
		tickets = append(tickets, kitchenTicket{
			ID: item.ID,
			Code: item.Code,
			Channel: item.Channel,
			Status: item.Status,
			CustomerName: item.CustomerName,
			TableName: item.TableName,
			Notes: item.Notes,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			TargetMinutes: target,
			Items: items,
		})
		counts[item.Status]++
	}

	writeJSON(w, 200, map[string]any{
		"items": tickets,
		"counts": counts,
		"channelOptions": orderChannels,
		"serverTime": time.Now().UTC(),
	})
}

func (a *API) updateKitchenTicketStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_status", "El estado solicitado no es válido.")
		return
	}
	in.Status = strings.TrimSpace(in.Status)
	if in.Status != "preparando" && in.Status != "listo" {
		fail(w, 400, "invalid_kitchen_status", "Cocina solo puede iniciar preparación o marcar una comanda como lista.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos actualizar la comanda.")
		return
	}
	defer tx.Rollback(r.Context())

	var current string
	err = tx.QueryRow(r.Context(), `
		SELECT status
		FROM orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos validar la comanda.")
		return
	}
	if !validKitchenTransition(current, in.Status) {
		fail(w, 409, "invalid_kitchen_transition", "La comanda cambió de estado. Actualiza la cola antes de continuar.")
		return
	}

	tag, err := tx.Exec(r.Context(), `
		UPDATE orders
		SET status=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		r.PathValue("id"), s.OrganizationID, s.LocationID, in.Status)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 503, "kitchen_unavailable", "No pudimos actualizar la comanda.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos actualizar la comanda.")
		return
	}
	action := "kitchen.started"
	if in.Status == "listo" {
		action = "kitchen.ready"
	}
	a.audit(r, action, "order", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}
