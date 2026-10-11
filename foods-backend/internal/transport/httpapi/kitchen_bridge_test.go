package httpapi

import (
	"context"
	orderdomain "github.com/foods-platform/foods-backend/internal/modules/orders/domain"

	"github.com/jackc/pgx/v5"
)

type kitchenTicket = orderdomain.KitchenTicket

var validKitchenTransition = orderdomain.ValidKitchenTransition
var normalizedKitchenRoundStatus = orderdomain.NormalizedKitchenRoundStatus

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
