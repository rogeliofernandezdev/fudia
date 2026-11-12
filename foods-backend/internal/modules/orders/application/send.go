package application

import (
	"context"
	"errors"

	noticecontracts "github.com/foods-platform/foods-backend/internal/modules/notifications/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// SendUnsent es «Enviar comanda»: pasa a Cocina/Barra lo guardado sin enviar
// (el primer pedido o las rondas que pidió el cliente por WhatsApp), deja a quien
// lo envía como responsable si aún no hay uno y avisa «en preparación» a cada
// WhatsApp que pidió lo enviado. Un pedido del personal sin productos de
// preparación conserva el envío clásico por estado.
func (a *Orders) SendUnsent(ctx context.Context, s scope, id string) (domain.Order, error) {
	var result domain.Order
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar la comanda.")
	}
	defer tx.Rollback(ctx)

	current, channel, _, _, completed, waiterID, err := a.repo.LockOrderStatus(tx.Context(), s, id)
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar la comanda.")
	}
	if !domain.CanManageOrderService(s.UserID, channel, waiterID) {
		return result, orderFailure(apperror.Forbidden, "order_assigned_to_another_waiter", "Esta mesa está siendo atendida por otro mozo. Solo puedes consultar su detalle.")
	}
	if completed || current == "cancelado" {
		return result, orderFailure(apperror.Conflict, "order_closed", "Esta atención ya está finalizada.")
	}
	pending, err := a.repo.HasUnsentItems(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la comanda.")
	}
	if !pending {
		if current == "nuevo" {
			_ = tx.Rollback(ctx)
			return a.SetStatus(ctx, s, id, OrderStatusInput{Status: "confirmado"})
		}
		return result, orderFailure(apperror.Conflict, "nothing_to_send", "No hay productos por enviar a cocina.")
	}
	recipients, err := a.repo.UnsentRecipients(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la comanda.")
	}
	if _, err = a.repo.SendUnsentItems(tx.Context(), s, id); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar los productos.")
	}
	if current == "nuevo" {
		if _, err = a.repo.SetStatus(tx.Context(), s, id, OrderStatusInput{Status: "confirmado"}); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar la comanda.")
		}
	}
	if err = a.repo.SyncServiceStatus(tx.Context(), s, id); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar la atención.")
	}
	if err = a.repo.AssignAttendant(tx.Context(), s, id, s.UserID); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar quién atiende.")
	}
	o, err := a.repo.Record(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el pedido actualizado.")
	}
	if err = a.paymentSummary(tx.Context(), s, &o); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el estado de cobro del pedido.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar la comanda.")
	}
	_ = a.audit.Record(ctx, s, "order.sent", "order", id)
	if a.notices != nil {
		for _, recipient := range recipients {
			if notice, ok := noticecontracts.OrderSentForCustomer(id, recipient.Phone, recipient.ConversationID, recipient.Key); ok {
				_ = a.notices.PublishCustomer(ctx, s, notice)
			}
		}
	}
	a.dispatchChanged(ctx, s, id)
	return o, nil
}
