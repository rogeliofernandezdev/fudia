package httpapi

import (
	"context"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type kitchenTicket = orderdomain.KitchenTicket

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
	var hasService bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2)`, orderID, s.OrganizationID).Scan(&hasService); err != nil {
		return err
	}
	if hasService {
		return syncOrderServiceStatus(ctx, tx, s, orderID)
	}
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
	a.handlers.Orders.ListKitchenTickets(w, r)
}

func (a *API) updateKitchenTicketStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Orders.UpdateKitchenTicketStatus(w, r)
}

// The caller holds the order lock. Check after locking so snapshots added by a
// concurrent consumption cannot be mistaken for a historical kitchen ticket.
func allowLegacyKitchenTicket(w http.ResponseWriter, r *http.Request, tx pgx.Tx, s scope, orderID string) bool {
	var hasService bool
	if err := tx.QueryRow(r.Context(), `SELECT EXISTS(
		SELECT 1 FROM order_service_items WHERE order_id=$1 AND organization_id=$2
	)`, orderID, s.OrganizationID).Scan(&hasService); err != nil {
		fail(w, 503, "kitchen_unavailable", "No pudimos validar la comanda.")
		return false
	}
	if hasService {
		fail(w, 404, "kitchen_ticket_not_found", "La comanda no existe en esta estación.")
		return false
	}
	return true
}
