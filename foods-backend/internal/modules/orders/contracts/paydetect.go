package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// Puerto de lectura para Fudia Pay: valida que un pedido exista dentro del
// alcance consultado antes de ligarle una notificación de pago detectado.
// La tenencia la resuelve el alcance de la sesión, nunca el dispositivo.

var ErrPayOrderNotFound = apperror.New(apperror.NotFound, "pay_order_not_found", "El pedido no existe en este local.")

type PayOrderRef struct {
	ID           string  `json:"id"`
	Code         string  `json:"code"`
	CustomerName string  `json:"customerName"`
	Total        float64 `json:"total"`
	Status       string  `json:"status"`
}

type PayOrderLookup interface {
	PayOrderRef(ctx context.Context, scope requestscope.Scope, orderID string) (PayOrderRef, error)
}
