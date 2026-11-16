package postgres

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/jackc/pgx/v5"
)

// Lectura estrecha para Fudia Pay: el pedido debe existir dentro del alcance
// de la sesión para que el cajero pueda ligarle una notificación de pago
// detectado. No muta pedidos.
var _ contracts.PayOrderLookup = (*Orders)(nil)

func (r *Orders) PayOrderRef(ctx context.Context, s scope, orderID string) (contracts.PayOrderRef, error) {
	var ref contracts.PayOrderRef
	var total float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT o.id::text, o.code, o.customer_name, o.total, o.status
		FROM orders o
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
	`, orderID, s.OrganizationID, s.LocationID).
		Scan(&ref.ID, &ref.Code, &ref.CustomerName, &total, &ref.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return contracts.PayOrderRef{}, contracts.ErrPayOrderNotFound
	}
	if err != nil {
		return contracts.PayOrderRef{}, err
	}
	ref.Total = total
	return ref, nil
}
