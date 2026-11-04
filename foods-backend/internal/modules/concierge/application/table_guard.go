package application

import (
	"context"

	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	servicecontracts "github.com/foods-platform/foods-backend/internal/modules/service/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// TableGuard decide si un QR de mesa puede abrir una atención nueva.
type TableGuard interface {
	GuardTable(ctx context.Context, token string) error
}

// GuardTable impide que un QR nuevo se sume a una cuenta anterior sin cerrar:
// si la comanda abierta de la mesa lleva horas sin rondas nuevas, avisa al
// personal y responde table_account_stale. Un grupo que pide dentro de ese
// tiempo sigue sumando a la misma cuenta. Si la validación falla, no bloquea:
// Orders vuelve a validar al recibir el pedido.
func (a *Concierge) GuardTable(ctx context.Context, token string) error {
	qr, err := a.resolve(ctx, token)
	if err != nil || qr.ServiceType == servicecontracts.ServiceTypeCounter {
		return nil
	}
	account, err := a.orders.StaleTableAccount(ctx, qr.Scope, qr.TableID)
	if err != nil || !account.Stale {
		return nil
	}
	a.notify(ctx, qr.Scope, noticecontracts.StaleAccount(qr.TableID, qr.TableName, account.OrderID, account.LastActivity))
	return conciergeFailure(apperror.Conflict, ordercontracts.StaleTableAccountCode, "La mesa tiene una cuenta anterior sin cerrar. Ya avisamos al personal.")
}

// warnStaleAccount avisa al personal cuando Orders rechaza un pedido por una
// cuenta anterior sin cerrar (la atención se abrió antes de quedar inactiva).
func (a *Concierge) warnStaleAccount(ctx context.Context, qr QRContext) {
	account, err := a.orders.StaleTableAccount(ctx, qr.Scope, qr.TableID)
	if err == nil && account.Stale {
		a.notify(ctx, qr.Scope, noticecontracts.StaleAccount(qr.TableID, qr.TableName, account.OrderID, account.LastActivity))
	}
}
