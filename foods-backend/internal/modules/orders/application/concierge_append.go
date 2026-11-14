package application

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

// appendConciergeDelivery adds only the confirmed new dishes. It never replaces
// cooked items, changes delivery, charges its fee again or starts another order.
func (a *Orders) appendConciergeDelivery(ctx context.Context, s scope, in ConciergeDeliveryInput) (TableSubmission, error) {
	var result TableSubmission
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	if err = a.repo.LockConciergeDeliveryRequest(tx.Context(), s, in.RequestID); err != nil {
		return result, err
	}
	if id, replayErr := a.repo.ConciergeDeliveryRequestOrder(tx.Context(), s, in.RequestID); replayErr == nil {
		if err = tx.Commit(ctx); err != nil {
			return result, err
		}
		out, readErr := a.conciergeOrderView(ctx, s, id)
		return TableSubmission{Order: out}, readErr
	} else if !errors.Is(replayErr, ErrNotFound) {
		return result, replayErr
	}
	o, err := a.repo.LockServiceOrder(tx.Context(), s, in.OrderID)
	if err != nil {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	active, err := a.repo.ConciergeOrderActive(tx.Context(), s, in.ConversationID, o.ID)
	if err != nil || !active || (o.Channel != "delivery" && o.Channel != "recojo") {
		return result, orderFailure(apperror.Conflict, "order_not_appendable", "El pedido no está activo en esta conversación.")
	}
	if o.Status != "nuevo" && o.Status != "confirmado" && o.Status != "preparando" {
		return result, orderFailure(apperror.Conflict, "order_not_appendable", "Tu pedido ya está listo o salió del local. Solicita al personal agregar más productos.")
	}
	if o.CompletedAt != "" || o.BillClosedAt != "" || (o.DeliveryDispatchStatus != "" && o.DeliveryDispatchStatus != "own" && o.DeliveryDispatchStatus != "unassigned") {
		return result, orderFailure(apperror.Conflict, "order_not_appendable", "El reparto o cobro de tu pedido ya está coordinado. Solicita al personal agregar más productos.")
	}
	paid, err := a.repo.NetPaid(tx.Context(), s, o.ID)
	if err != nil {
		return result, err
	}
	if paid > 0 {
		return result, orderFailure(apperror.Conflict, "paid_order_not_editable", "Tu pedido ya tiene pagos registrados. Solicita al personal agregar más productos.")
	}
	if in.Channel != o.Channel || (o.Channel == "delivery" && in.Address != o.Address) {
		return result, orderFailure(apperror.Conflict, "delivery_details_changed", "Para cambiar la forma o dirección de entrega de este pedido, solicita ayuda al personal.")
	}
	current, _ := strconv.ParseFloat(o.Total, 64)
	expected, _ := strconv.ParseFloat(in.ExpectedOrderTotal, 64)
	if math.Abs(current-expected) >= 0.005 {
		return result, orderFailure(apperror.Conflict, "order_changed", "El pedido cambió. Revisa el resumen actualizado antes de confirmar los productos adicionales.")
	}
	prepared, added, err := a.prepareItems(tx.Context(), s, "", in.Items)
	if err != nil {
		return result, err
	}
	if err = a.ensureRound(tx.Context(), s, o.ID, o.Status); err != nil {
		return result, err
	}
	roundID, _, err := a.repo.CreateKitchenRound(tx.Context(), s, o.ID, "confirmado", "concierge")
	if err != nil {
		return result, err
	}
	if _, err = a.repo.IdentifyRound(tx.Context(), s, roundID, AppendInput{RequestKey: in.RequestID}); err != nil {
		return result, err
	}
	if err = a.repo.InsertItems(tx.Context(), s, o.ID, roundID, prepared); err != nil {
		return result, err
	}
	if _, err = a.repo.HoldRoundItems(tx.Context(), s, o.ID, roundID); err != nil {
		return result, err
	}
	if err = a.stock.ApplyOrderQuantityDelta(tx.Context(), s, o.ID, domain.PreparedQuantityUsage(prepared), "sale"); err != nil {
		return result, err
	}
	recipeUsage, err := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(prepared))
	if err != nil {
		return result, err
	}
	if err = a.stock.ApplyRecipeUsageDelta(tx.Context(), s, o.ID, recipeUsage); err != nil {
		return result, err
	}
	if _, err = a.repo.AddOrderTotal(tx.Context(), s, o, added); err != nil {
		return result, err
	}
	if err = a.syncKitchenStatus(tx.Context(), s, o.ID); err != nil {
		return result, err
	}
	if err = a.repo.SaveConciergeDeliveryRequest(tx.Context(), s, in, o.ID); err != nil {
		return result, err
	}
	if err = a.audit.Record(tx.Context(), s, "concierge.delivery_order.items_added", "order", o.ID); err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return result, err
	}
	out, err := a.conciergeOrderView(ctx, s, o.ID)
	return TableSubmission{Order: out}, err
}
