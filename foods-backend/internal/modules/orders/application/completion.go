package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

func (a *Orders) CompletePaid(ctx context.Context, s scope, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos finalizar el pedido.")
	}
	defer tx.Rollback(ctx)
	var total, paid float64
	var status string
	var salon bool
	err = func() error {
		var err error
		total, status, salon, err = a.repo.LockPaidCompletion(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	if err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el pedido.")
	}
	if err = func() error { var err error; paid, err = a.repo.CompletionPaid(tx.Context(), s, id); return err }(); err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar el saldo del pedido.")
	}
	if status == "cancelado" {
		return orderFailure(apperror.Conflict, "order_not_completable", "Un pedido cancelado no puede finalizarse.")
	}
	if paid+0.00001 < total {
		return orderFailure(apperror.Conflict, "payment_incomplete", "El pedido todavía tiene saldo pendiente.")
	}
	if salon {
		var closed bool
		closed, err = a.repo.CompleteDeliveredAccount(tx.Context(), s, id)
		if err != nil {
			return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos finalizar la cuenta.")
		}
		if !closed {
			return orderFailure(apperror.Conflict, "delivery_required", "Cierra la cuenta y completa las entregas antes de finalizar.")
		}
		if err = tx.Commit(ctx); err != nil {
			return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos finalizar.")
		}
		_ = a.audit.Record(ctx, s, "order.completed", "order", id)
		return nil
	}
	if status != "listo" && status != "en_camino" {
		return orderFailure(apperror.Conflict, "order_not_ready", "El pago está completo, pero la comanda todavía no está lista para entregarse.")
	}
	var pending bool
	if err = func() error { var err error; pending, err = a.repo.CompletionPending(tx.Context(), s, id); return err }(); err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos validar la entrega.")
	}
	if pending {
		return orderFailure(apperror.Conflict, "items_not_ready", "Todavía hay productos en preparación.")
	}
	if _, err = a.repo.DeliverPaidItems(tx.Context(), s, id); err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos registrar la entrega.")
	}
	if _, err = a.repo.CompletePaidOrder(tx.Context(), s, id); err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos liberar la mesa.")
	}
	if err = tx.Commit(ctx); err != nil {
		return orderFailure(apperror.Unavailable, "payments_unavailable", "No pudimos finalizar el pedido.")
	}
	_ = a.audit.Record(ctx, s, "order.completed", "order", id)
	return nil

}
