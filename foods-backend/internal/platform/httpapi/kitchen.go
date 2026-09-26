package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type kitchenTicket struct {
	ID            string      `json:"id"`
	OrderID       string      `json:"orderId"`
	RoundNumber   int         `json:"roundNumber"`
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

func normalizedKitchenRoundStatus(status string) string {
	switch status {
	case "confirmado", "preparando", "listo":
		return status
	default:
		return "confirmado"
	}
}

func createKitchenRound(ctx context.Context, tx pgx.Tx, s scope, orderID, status, source string) (string, int, error) {
	var sequence int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(sequence),0)+1
		FROM order_kitchen_rounds
		WHERE order_id=$1 AND organization_id=$2 AND location_id=$3
	`, orderID, s.OrganizationID, s.LocationID).Scan(&sequence); err != nil {
		return "", 0, err
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO order_kitchen_rounds(
		  organization_id,location_id,order_id,sequence,status,source
		)
		VALUES($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, s.OrganizationID, s.LocationID, orderID, sequence, normalizedKitchenRoundStatus(status), source).Scan(&id)
	return id, sequence, err
}

func ensureKitchenRoundForOrder(ctx context.Context, tx pgx.Tx, s scope, orderID, currentStatus string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
		  SELECT 1 FROM order_kitchen_rounds
		  WHERE order_id=$1 AND organization_id=$2 AND location_id=$3
		)
	`, orderID, s.OrganizationID, s.LocationID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	roundID, _, err := createKitchenRound(ctx, tx, s, orderID, currentStatus, "order")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE order_items
		SET kitchen_round_id=$3
		WHERE order_id=$1 AND organization_id=$2
		  AND kitchen_round_id IS NULL
	`, orderID, s.OrganizationID, roundID)
	return err
}

func syncOrderKitchenStatus(ctx context.Context, tx pgx.Tx, s scope, orderID string) error {
	var confirmed, preparing, ready int
	if err := tx.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE status='confirmado'),
		  count(*) FILTER (WHERE status='preparando'),
		  count(*) FILTER (WHERE status='listo')
		FROM order_kitchen_rounds
		WHERE order_id=$1 AND organization_id=$2 AND location_id=$3
	`, orderID, s.OrganizationID, s.LocationID).Scan(&confirmed, &preparing, &ready); err != nil {
		return err
	}
	if confirmed+preparing+ready == 0 {
		return nil
	}
	next := "listo"
	if confirmed > 0 {
		next = "confirmado"
	}
	if preparing > 0 {
		next = "preparando"
	}
	_, err := tx.Exec(ctx, `
		UPDATE orders
		SET status=$4,updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		  AND status NOT IN ('entregado','cancelado')
	`, orderID, s.OrganizationID, s.LocationID, next)
	return err
}

func (a *API) kitchenTargetMinutes(r *http.Request, orderID, organizationID string) (*int, error) {
	return a.kitchenTargetMinutesForRound(r, orderID, organizationID, "")
}

func (a *API) kitchenTargetMinutesForRound(r *http.Request, orderID, organizationID, roundID string) (*int, error) {
	var target int
	err := a.db.QueryRow(r.Context(), `
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
		) preparation_times`, orderID, organizationID, roundID).Scan(&target)
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

	tickets := []kitchenTicket{}
	counts := map[string]int{"confirmado": 0, "preparando": 0, "listo": 0}

	legacyRows, err := a.db.Query(r.Context(), `
		SELECT `+orderColumns+`
		FROM orders
		WHERE organization_id=$1
		  AND location_id=$2
		  AND status IN ('confirmado','preparando','listo')
		  AND ($3='' OR channel=$3)
		  AND NOT EXISTS (
		    SELECT 1 FROM order_kitchen_rounds kr
		    WHERE kr.order_id=orders.id
		      AND kr.organization_id=orders.organization_id
		      AND kr.location_id=orders.location_id
		  )
		ORDER BY
		  CASE status WHEN 'preparando' THEN 0 WHEN 'confirmado' THEN 1 ELSE 2 END,
		  updated_at ASC,created_at ASC
		LIMIT 150`, s.OrganizationID, s.LocationID, channel)
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
		return
	}
	for legacyRows.Next() {
		item, scanErr := scanOrder(legacyRows)
		if scanErr != nil {
			legacyRows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos cargar las comandas de cocina.")
			return
		}
		items, itemsErr := loadOrderItems(r.Context(), a.db, item.ID, s.OrganizationID)
		if itemsErr != nil {
			legacyRows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos cargar los productos de las comandas.")
			return
		}
		target, targetErr := a.kitchenTargetMinutes(r, item.ID, s.OrganizationID)
		if targetErr != nil {
			legacyRows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos calcular los tiempos de preparación.")
			return
		}
		tickets = append(tickets, kitchenTicket{
			ID: item.ID, OrderID: item.ID, Code: item.Code, Channel: item.Channel,
			Status: item.Status, CustomerName: item.CustomerName, TableName: item.TableName,
			Notes: item.Notes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
			TargetMinutes: target, Items: items,
		})
		counts[item.Status]++
	}
	if err = legacyRows.Err(); err != nil {
		legacyRows.Close()
		fail(w, 503, "kitchen_unavailable", "No pudimos completar las comandas de cocina.")
		return
	}
	legacyRows.Close()

	roundRows, err := a.db.Query(r.Context(), `
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
		  AND ($3='' OR o.channel=$3)
		ORDER BY
		  CASE kr.status WHEN 'preparando' THEN 0 WHEN 'confirmado' THEN 1 ELSE 2 END,
		  kr.updated_at ASC,kr.created_at ASC
		LIMIT 150
	`, s.OrganizationID, s.LocationID, channel)
	if err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos cargar las rondas de cocina.")
		return
	}
	for roundRows.Next() {
		var ticket kitchenTicket
		var baseCode string
		if err := roundRows.Scan(
			&ticket.ID, &ticket.RoundNumber, &ticket.Status,
			&ticket.OrderID, &baseCode, &ticket.Channel, &ticket.CustomerName,
			&ticket.TableName, &ticket.Notes, &ticket.CreatedAt, &ticket.UpdatedAt,
		); err != nil {
			roundRows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos leer las rondas de cocina.")
			return
		}
		ticket.Code = fmt.Sprintf("%s · R%d", baseCode, ticket.RoundNumber)
		ticket.Items, err = loadOrderItemsForRound(r.Context(), a.db, ticket.OrderID, s.OrganizationID, ticket.ID)
		if err != nil {
			roundRows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos cargar los productos de una ronda.")
			return
		}
		ticket.TargetMinutes, err = a.kitchenTargetMinutesForRound(r, ticket.OrderID, s.OrganizationID, ticket.ID)
		if err != nil {
			roundRows.Close()
			fail(w, 503, "kitchen_unavailable", "No pudimos calcular el tiempo de una ronda.")
			return
		}
		tickets = append(tickets, ticket)
		counts[ticket.Status]++
	}
	if err = roundRows.Err(); err != nil {
		roundRows.Close()
		fail(w, 503, "kitchen_unavailable", "No pudimos completar las rondas de cocina.")
		return
	}
	roundRows.Close()

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

	var roundCurrent, roundOrderID string
	err = tx.QueryRow(r.Context(), `
		SELECT kr.status,kr.order_id::text
		FROM order_kitchen_rounds kr
		JOIN orders o ON o.id=kr.order_id AND o.organization_id=kr.organization_id
		WHERE kr.id=$1 AND kr.organization_id=$2 AND kr.location_id=$3
		  AND o.status NOT IN ('entregado','cancelado')
		FOR UPDATE OF kr,o
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&roundCurrent, &roundOrderID)
	if err == nil {
		if !validKitchenTransition(roundCurrent, in.Status) {
			fail(w, 409, "invalid_kitchen_transition", "La ronda cambió de estado. Actualiza la cola antes de continuar.")
			return
		}
		if _, err = tx.Exec(r.Context(), `
			UPDATE order_kitchen_rounds
			SET status=$4,updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
		`, r.PathValue("id"), s.OrganizationID, s.LocationID, in.Status); err != nil {
			fail(w, 503, "kitchen_unavailable", "No pudimos actualizar la ronda.")
			return
		}
		if err = syncOrderKitchenStatus(r.Context(), tx, s, roundOrderID); err != nil {
			fail(w, 503, "kitchen_unavailable", "No pudimos actualizar el estado general de la comanda.")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			fail(w, 503, "kitchen_unavailable", "No pudimos confirmar el cambio de la ronda.")
			return
		}
		action := "kitchen.round.started"
		if in.Status == "listo" {
			action = "kitchen.round.ready"
		}
		a.audit(r, action, "order_round", r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		fail(w, 503, "kitchen_unavailable", "No pudimos validar la ronda.")
		return
	}

	var current string
	err = tx.QueryRow(r.Context(), `
		SELECT status
		FROM orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE
	`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "order_not_found", "El pedido o ronda no existe.")
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
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
	`, r.PathValue("id"), s.OrganizationID, s.LocationID, in.Status)
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
