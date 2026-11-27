package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// ReservationDeposits es el puerto de Reservas hacia la garantía cobrada en
// Caja. Refund y Retain corren dentro de la transacción de quien llama; la
// boleta de una garantía retenida se pide después del commit.
type ReservationDeposits interface {
	HeldDeposit(ctx context.Context, s requestscope.Scope, reservationID string) (bool, error)
	RefundHeldDeposit(ctx context.Context, s requestscope.Scope, reservationID string) error
	RetainHeldDeposit(ctx context.Context, s requestscope.Scope, reservationID string) (string, error)
	IssueRetainedReceipt(ctx context.Context, s requestscope.Scope, orderID string)
}
