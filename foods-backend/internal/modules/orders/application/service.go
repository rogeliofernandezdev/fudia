package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

func (a *Orders) lockService(ctx context.Context, s scope, id string) (domain.Order, error) {
	o, err := a.repo.LockServiceOrder(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return o, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	if err != nil {
		return o, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos consultar la atención.")
	}
	if !domain.CanManageOrderService(s.UserID, o.Channel, o.WaiterID) {
		return o, orderFailure(apperror.Forbidden, "order_assigned_to_another_waiter", "Esta mesa está siendo atendida por otro mozo. Solo puedes consultar su detalle.")
	}
	if o.CompletedAt != "" || o.Status == "cancelado" {
		return o, orderFailure(apperror.Conflict, "order_closed", "Esta atención ya está finalizada.")
	}
	return o, nil
}
func (a *Orders) ensureRound(ctx context.Context, s scope, id, status string) error {
	exists, err := a.repo.HasKitchenRounds(ctx, s, id)
	if err != nil || exists {
		return err
	}
	round, _, err := a.repo.CreateKitchenRound(ctx, s, id, domain.NormalizedKitchenRoundStatus(status), "order")
	if err != nil {
		return err
	}
	_, err = a.repo.AssignOrphanItems(ctx, s, id, round)
	return err
}
func (a *Orders) RequireDelivered(ctx context.Context, s scope, orderID, status string) error {
	pending, err := a.repo.UndeliveredItems(ctx, s, orderID)
	if err != nil {
		return orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos verificar la entrega de los productos.")
	}
	if status != "entregado" || pending {
		return orderFailure(apperror.Conflict, "products_not_delivered", "Entrega todos los productos antes de cerrar la cuenta o cobrar.")
	}
	return nil
}
func (a *Orders) CloseBill(ctx context.Context, s scope, id string) (domain.Order, error) {
	var result domain.Order
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cerrar la cuenta.")
	}
	defer tx.Rollback(ctx)
	o, err := a.lockService(tx.Context(), s, id)
	if err != nil {
		return result, err
	}
	if o.Channel != "salon" {
		return result, orderFailure(apperror.Conflict, "table_account_required", "Esta acción corresponde a una cuenta de Salón.")
	}
	if o.Status == "nuevo" {
		return result, orderFailure(apperror.Conflict, "order_not_submitted", "Registra el pedido antes de cerrar la cuenta.")
	}
	if err := a.RequireDelivered(tx.Context(), s, o.ID, o.Status); err != nil {
		return result, err
	}
	_, err = a.repo.CloseBill(tx.Context(), s, o)
	closed := false
	if err == nil {
		closed, err = a.repo.CompleteDeliveredAccount(tx.Context(), s, o.ID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cerrar la cuenta.")
	}
	_ = a.audit.Record(ctx, s, "order.bill.closed", "order", o.ID)
	if closed {
		_ = a.audit.Record(ctx, s, "order.completed", "order", o.ID)
	}
	return a.Get(ctx, s, o.ID)

}
func (a *Orders) DeliverItem(ctx context.Context, s scope, id string, itemID string) (domain.Order, error) {
	var result domain.Order
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar la entrega.")
	}
	defer tx.Rollback(ctx)
	o, err := a.lockService(tx.Context(), s, id)
	if err != nil {
		return result, err
	}
	if o.Channel != "salon" {
		return result, orderFailure(apperror.Conflict, "table_account_required", "La entrega por producto corresponde a la atención de Salón.")
	}
	tag, err := a.repo.DeliverItem(tx.Context(), s, o, itemID)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar la entrega.")
	}
	if tag == 0 {
		return result, orderFailure(apperror.Conflict, "item_not_ready", "El producto no existe en esta cuenta o todavía no está listo para entregar.")
	}
	closed := false
	if err = a.repo.SyncServiceStatus(tx.Context(), s, o.ID); err == nil {
		closed, err = a.repo.CompleteDeliveredAccount(tx.Context(), s, o.ID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar la entrega.")
	}
	_ = a.audit.Record(ctx, s, "order.item.delivered", "order", o.ID)
	if closed {
		_ = a.audit.Record(ctx, s, "order.completed", "order", o.ID)
	}
	return a.Get(ctx, s, o.ID)

}
func (a *Orders) AppendItems(ctx context.Context, s scope, id string, in AppendInput) (domain.Order, error) {
	var result domain.Order
	if len(in.Items) == 0 || len(in.Items) > 200 || len(strings.TrimSpace(in.RequestKey)) < 1 || len(in.RequestKey) > 160 {
		return result, orderFailure(apperror.Invalid, "invalid_order", "Agrega productos y un identificador de envío válido.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos agregar productos.")
	}
	defer tx.Rollback(ctx)
	o, err := a.lockService(tx.Context(), s, id)
	if err != nil {
		return result, err
	}
	if o.Channel != "salon" || o.Status == "nuevo" {
		return result, orderFailure(apperror.Conflict, "order_not_appendable", "Registra primero el pedido de mesa.")
	}
	var exists bool
	if err = func() error {
		var err error
		exists, err = a.repo.AppendAlreadyExists(tx.Context(), s, o, in)
		return err
	}(); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el envío.")
	}
	if exists {
		if err = tx.Commit(ctx); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos consultar el envío.")
		}
		return a.Get(ctx, s, o.ID)
	}
	if o.BillClosedAt != "" {
		return result, orderFailure(apperror.Conflict, "order_bill_closed", "La cuenta está por cobrar. No se pueden agregar productos después de cerrarla.")
	}
	var paid bool
	if err = func() error { var err error; paid, err = a.repo.AccountPaid(tx.Context(), s, o); return err }(); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el saldo.")
	}
	if paid {
		return result, orderFailure(apperror.Conflict, "paid_order_not_editable", "La cuenta ya está pagada. El nuevo consumo necesita una nueva atención.")
	}
	if err = a.ensureRound(tx.Context(), s, o.ID, o.Status); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos conservar las comandas anteriores.")
	}
	roundID, _, err := a.repo.CreateKitchenRound(tx.Context(), s, o.ID, "confirmado", "order")
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la nueva comanda.")
	}
	if _, err = a.repo.IdentifyRound(tx.Context(), s, roundID, in); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos identificar el envío.")
	}
	items, added, prepErr := a.prepareItems(tx.Context(), s, "", in.Items)
	if prepErr != nil {
		return result, prepErr
	}
	// New lines must enter Pending even when the previous domain.Order was Delivered.
	if _, err = a.repo.ReopenForAppend(tx.Context(), s, o); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos abrir la comanda.")
	}
	if err = a.repo.InsertItems(tx.Context(), s, o.ID, roundID, items); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos agregar los productos.")
	}
	if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, o.ID, domain.PreparedQuantityUsage(items), "sale"); quantityErr != nil {
		return result, quantityErr
	}
	recipeUsage, recipeErr := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(items))
	if recipeErr != nil {
		return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar los insumos.")
	}
	if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, o.ID, recipeUsage); quantityErr != nil {
		return result, quantityErr
	}
	_, err = a.repo.AddOrderTotal(tx.Context(), s, o, added)
	if err == nil {
		err = a.repo.SyncServiceStatus(tx.Context(), s, o.ID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar los productos adicionales.")
	}
	_ = a.audit.Record(ctx, s, "order.items.added", "order", o.ID)
	return a.Get(ctx, s, o.ID)

}

type AppendInput struct {
	Items      []domain.OrderItemInput `json:"items"`
	RequestKey string                  `json:"requestKey"`
}
