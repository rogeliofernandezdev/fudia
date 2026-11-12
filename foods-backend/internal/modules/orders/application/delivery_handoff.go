package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// These methods participate in Delivery's unit of work. Composition supplies
// the same repositories/pool: no nested transaction or post-commit money move.
func (a *Orders) ReleaseDeliveryFeeInTransaction(ctx context.Context, s scope, id string) error {
	_, _, _, _, completed, _, err := a.repo.LockOrderStatus(ctx, s, id)
	if err != nil {
		return err
	}
	paid, err := a.repo.NetPaid(ctx, s, id)
	if err != nil {
		return err
	}
	if completed || paid > 0.00001 {
		return orderFailure(apperror.Conflict, "delivery_payment_conflict", "El pedido ya tiene pagos o está cerrado. Coordina el reparto con el local.")
	}
	changed, err := a.repo.ReleaseDeliveryFee(ctx, s, id)
	if err != nil {
		return err
	}
	if changed > 0 {
		return a.audit.Record(ctx, s, "order.delivery_fee_to_partner", "order", id)
	}
	return nil
}

func (a *Orders) RestoreDeliveryFeeInTransaction(ctx context.Context, s scope, id string, fee float64) error {
	_, _, _, _, completed, _, err := a.repo.LockOrderStatus(ctx, s, id)
	if err != nil {
		return err
	}
	if completed {
		return orderFailure(apperror.Conflict, "order_closed", "El pedido ya está cerrado.")
	}
	paid, err := a.repo.NetPaid(ctx, s, id)
	if err != nil {
		return err
	}
	if paid > 0.00001 {
		return orderFailure(apperror.Conflict, "delivery_payment_conflict", "Devuelve primero los pagos registrados antes de cambiar el envío.")
	}
	changed, err := a.repo.RestoreDeliveryFee(ctx, s, id, fee)
	if err != nil {
		return err
	}
	if changed > 0 {
		return a.audit.Record(ctx, s, "order.delivery_fee_restored", "order", id)
	}
	return nil
}

func (a *Orders) MarkDeliveryOnTheWayInTransaction(ctx context.Context, s scope, id string) error {
	status, channel, _, _, completed, _, err := a.repo.LockOrderStatus(ctx, s, id)
	if err != nil {
		return err
	}
	if channel != "delivery" || completed {
		return orderFailure(apperror.Conflict, "order_closed", "El pedido ya está cerrado.")
	}
	if status == "en_camino" || status == "entregado" {
		return nil
	}
	if status != "listo" {
		return orderFailure(apperror.Conflict, "delivery_not_ready", "El pedido aún no está listo para recoger.")
	}
	if _, err = a.repo.SetStatus(ctx, s, id, OrderStatusInput{Status: "en_camino"}); err != nil {
		return err
	}
	if err = a.audit.Record(ctx, s, "status_updated", "order", id); err != nil {
		return err
	}
	o, err := a.repo.Record(ctx, s, id)
	if err != nil {
		return err
	}
	a.noticeOnTheWay(ctx, s, o)
	return nil
}
