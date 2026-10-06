package httpapi

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// The caller holds the order row lock: payment and delivery share one atomic
// closure, even when a cashier and a waiter act concurrently.
func completeDeliveredSalonOrder(ctx context.Context, tx pgx.Tx, s scope, orderID string) (bool, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE orders o
		SET completed_at=now(),updated_at=now()
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3
		  AND o.channel='salon' AND o.status='entregado' AND o.completed_at IS NULL
		  AND `+netPaidSQL+` >= o.total
	`, orderID, s.OrganizationID, s.LocationID)
	return tag.RowsAffected() > 0, err
}
