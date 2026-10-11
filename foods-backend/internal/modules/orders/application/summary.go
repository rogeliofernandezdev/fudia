package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

func (a *Orders) paymentSummary(ctx context.Context, s scope, o *domain.Order) error {
	paid, err := a.repo.NetPaid(ctx, s, o.ID)
	if err != nil {
		return err
	}
	return domain.ApplyPaymentSummary(o, paid)
}
