package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// Puerto para Fudia Pay: el dinero detectado y asociado por el cajero se
// registra como cobro real con el flujo existente del módulo de pagos; nunca
// con una segunda lógica de caja. La tenencia la resuelve el alcance de la
// sesión.

type PayRegisterInput struct {
	OrderID   string
	Method    string
	Amount    float64
	Reference string
}

type PayRegisterResult struct {
	PaymentID string
}

type PayRegisterer interface {
	PayRegister(ctx context.Context, scope requestscope.Scope, in PayRegisterInput) (PayRegisterResult, error)
}
