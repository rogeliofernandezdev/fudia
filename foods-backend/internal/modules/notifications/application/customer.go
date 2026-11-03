package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// CustomerClaimLimit acota cuántos avisos reclama Concierge por consulta.
const CustomerClaimLimit = 50

// PublishCustomer implementa contracts.Publisher: un aviso inválido se descarta
// y el mismo pedido avisa una sola vez por tipo.
func (a *Notices) PublishCustomer(ctx context.Context, s scope, in domain.CustomerInput) error {
	if s.OrganizationID == "" || s.LocationID == "" || in.Kind != domain.CustomerOrderReady ||
		in.OrderID == "" || in.CustomerPhone == "" || in.ConversationID == "" {
		return nil
	}
	return a.repo.InsertCustomer(ctx, s, in)
}

// ClaimCustomer reserva avisos pendientes para Concierge. Un reclamo no
// confirmado vence y vuelve a entregarse; Concierge deduplica por id.
func (a *Notices) ClaimCustomer(ctx context.Context, limit int) ([]domain.CustomerNotice, error) {
	if limit < 1 || limit > CustomerClaimLimit {
		limit = 20
	}
	out, err := a.repo.ClaimCustomer(ctx, limit)
	if err != nil {
		return nil, failure(apperror.Unavailable, "customer_notices_unavailable", "No pudimos cargar los avisos para clientes.")
	}
	return out, nil
}

// MarkCustomerDispatched confirma que Concierge ya encoló el aviso; repetirlo no cambia nada.
func (a *Notices) MarkCustomerDispatched(ctx context.Context, id string) error {
	if !uuidPattern.MatchString(id) {
		return failure(apperror.NotFound, "customer_notice_not_found", "El aviso no existe.")
	}
	err := a.repo.MarkCustomerDispatched(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return failure(apperror.NotFound, "customer_notice_not_found", "El aviso no existe.")
	}
	if err != nil {
		return failure(apperror.Unavailable, "customer_notices_unavailable", "No pudimos confirmar el aviso.")
	}
	return nil
}
