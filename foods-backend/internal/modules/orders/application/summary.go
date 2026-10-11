package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

func (a *Orders) paymentSummary(ctx context.Context, s scope, o *domain.Order) error {
	paid, err := a.repo.NetPaid(ctx, s, o.ID)
	if err != nil {
		return err
	}
	return domain.ApplyPaymentSummary(o, paid)
}

func (a *Orders) CompleteDeliveredAccount(ctx context.Context, s scope, id string) (bool, error) {
	return a.repo.CompleteDeliveredAccount(ctx, s, id)
}
func (a *Orders) PaymentOrder(ctx context.Context, s scope, id string) (domain.Order, error) {
	o, err := a.repo.Record(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return o, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	if err != nil {
		return o, orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos cargar el pedido.")
	}
	o.Items, err = a.repo.Items(ctx, s, id, "")
	if err != nil {
		return o, orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos cargar el detalle del pedido.")
	}
	return o, nil
}
