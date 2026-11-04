package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

const (
	StaleTableAccountCode    = contracts.StaleTableAccountCode
	StaleTableAccountMessage = "La mesa tiene una cuenta anterior sin cerrar. Pide ayuda al personal."
)

// StaleTableAccount indica si la comanda abierta de la mesa lleva más de
// domain.TableAccountIdleLimitSeconds sin rondas nuevas. Sin comanda abierta no
// hay cuenta anterior que proteger.
func (a *Orders) StaleTableAccount(ctx context.Context, s scope, tableID string) (contracts.StaleTableAccount, error) {
	orderID, lastActivity, stale, err := a.repo.StaleTableAccount(ctx, s, tableID, domain.TableAccountIdleLimitSeconds)
	if errors.Is(err, ErrNotFound) {
		return contracts.StaleTableAccount{}, nil
	}
	if err != nil {
		return contracts.StaleTableAccount{}, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la cuenta de la mesa.")
	}
	return contracts.StaleTableAccount{OrderID: orderID, LastActivity: lastActivity, Stale: stale}, nil
}
