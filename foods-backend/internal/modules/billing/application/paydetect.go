package application

import (
	"context"

	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
)

// PayRegister conecta Fudia Pay con el flujo de cobros existente: el pago
// detectado y asociado se registra con el mismo Create de pagos (turno,
// movimientos de caja, comprobante y cierre de pedido), nunca con una segunda
// lógica de caja.
var _ billingcontracts.PayRegisterer = (*Payments)(nil)

func (a *Payments) PayRegister(ctx context.Context, s scope, in billingcontracts.PayRegisterInput) (billingcontracts.PayRegisterResult, error) {
	view, err := a.Create(ctx, s, domain.PaymentInput{
		OrderID:   in.OrderID,
		Method:    in.Method,
		Amount:    in.Amount,
		Reference: in.Reference,
	})
	if err != nil {
		return billingcontracts.PayRegisterResult{}, err
	}
	return billingcontracts.PayRegisterResult{PaymentID: view.ID}, nil
}
