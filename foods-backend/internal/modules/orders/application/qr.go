package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strconv"
	"strings"
)

type QROrderInput struct {
	CustomerName   string                  `json:"customerName"`
	CustomerPhone  string                  `json:"customerPhone"`
	ConversationID string                  `json:"conversationId"`
	RequestID      string                  `json:"requestId"`
	Notes          string                  `json:"notes"`
	Items          []domain.OrderItemInput `json:"items"`
}
type TableSubmission struct {
	Order   domain.Order
	Created bool
}

func (a *Orders) SubmitTable(ctx context.Context, s scope, tableID string, token string, in QROrderInput) (TableSubmission, error) {
	var result TableSubmission
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.RequestID = strings.TrimSpace(in.RequestID)
	if in.RequestID == "" {
		in.RequestID = in.ConversationID
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if len(in.Items) == 0 || len(in.Items) > 50 || len(in.CustomerPhone) > 32 ||
		in.ConversationID == "" || len(in.ConversationID) > 160 || in.RequestID == "" || len(in.RequestID) > 160 || len(in.Notes) > 240 {
		return result, orderFailure(apperror.Invalid, "invalid_order", "Revisa los productos y datos del pedido.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos iniciar el pedido.")
	}
	defer tx.Rollback(ctx)

	var lockedTableID string
	_ = lockedTableID
	err = func() error {
		var err error
		lockedTableID, err = a.repo.LockQRTable(tx.Context(), s, tableID, token)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "table_not_found", "La mesa no existe o el QR ya no está activo.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la mesa.")
	}

	var existingOrderID string
	err = func() error {
		var err error
		existingOrderID, err = a.repo.QRRequestOrder(tx.Context(), s, tableID, in)
		return err
	}()
	if err == nil {
		var existing domain.Order
		existing, err = a.repo.Record(tx.Context(), s, existingOrderID)
		if err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar el pedido confirmado.")
		}
		existing.Items, err = a.repo.Items(tx.Context(), s, existing.ID, "")
		if err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar los productos del pedido.")
		}
		for _, item := range existing.Items {
			if qty, parseErr := strconv.ParseFloat(item.Qty, 64); parseErr == nil {
				existing.ItemCount += int(qty)
			}
		}
		if err = a.paymentSummary(tx.Context(), s, &existing); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar el estado del pedido.")
		}
		return TableSubmission{Order: existing}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la idempotencia del pedido.")
	}

	var openOrderID, openStatus, openChannel string
	var openCreatedBy *string
	var openSubtotal, openDeliveryFee float64
	var conciergeOwned bool
	var openBillClosed bool
	err = func() error {
		var err error
		openOrderID, openStatus, openChannel, openSubtotal, openDeliveryFee, openCreatedBy, openBillClosed, conciergeOwned, err = a.repo.LockQRTableOrder(tx.Context(), s, tableID)
		return err
	}()
	if err == nil {
		if openChannel == "whatsapp" {
			if openCreatedBy != nil || !conciergeOwned {
				return result, orderFailure(apperror.Conflict, "table_occupied", "La mesa tiene una comanda abierta de WhatsApp que no pertenece a Fudia Concierge.")
			}
		} else if openChannel != "salon" {
			return result, orderFailure(apperror.Conflict, "table_occupied", "La mesa tiene una comanda abierta gestionada por otro canal.")
		}
		if openChannel == "salon" && openBillClosed {
			return result, orderFailure(apperror.Conflict, "order_bill_closed", "La cuenta está por cobrar. No se pueden agregar productos después de cerrarla.")
		}
		if openStatus != "confirmado" && openStatus != "preparando" && openStatus != "listo" && openStatus != "entregado" {
			return result, orderFailure(apperror.Conflict, "concierge_order_not_editable", "La comanda ya no admite nuevas rondas desde Concierge.")
		}
		paid, paidErr := a.repo.NetPaid(tx.Context(), s, openOrderID)
		if paidErr != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar los pagos de la comanda.")
		}
		var total float64
		if err = func() error {
			var err error
			total, err = a.repo.QROrderTotal(tx.Context(), s, openOrderID)
			return err
		}(); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el saldo de la comanda.")
		}
		if paid > 0.00001 && paid+0.00001 >= total {
			return result, orderFailure(apperror.Conflict, "paid_order_not_editable", "La cuenta ya está pagada y no admite consumo adicional.")
		}

		if err = a.ensureRound(tx.Context(), s, openOrderID, openStatus); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos preparar la trazabilidad de las rondas de cocina.")
		}
		roundID, _, roundErr := a.repo.CreateKitchenRound(tx.Context(), s, openOrderID, "confirmado", "concierge")
		if roundErr != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos abrir la nueva ronda de cocina.")
		}
		prepared, addedSubtotal, preparationErr := a.prepareItems(tx.Context(), s, "", in.Items)
		if preparationErr != nil {
			return result, preparationErr
		}
		if err = a.repo.InsertItems(tx.Context(), s, openOrderID, roundID, prepared); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos agregar los productos a la comanda.")
		}
		if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, openOrderID, domain.PreparedQuantityUsage(prepared), "sale"); quantityErr != nil {
			return result, quantityErr
		}
		recipeUsage, recipeErr := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(prepared))
		if recipeErr != nil {
			return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el consumo de recetas.")
		}
		if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, openOrderID, recipeUsage); quantityErr != nil {
			return result, quantityErr
		}
		nextSubtotal := openSubtotal + addedSubtotal
		nextTotal := nextSubtotal + openDeliveryFee
		if _, err = a.repo.UpdateQRSubtotal(tx.Context(), s, openOrderID, nextSubtotal, nextTotal); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el total de la comanda.")
		}
		if err = a.syncKitchenStatus(tx.Context(), s, openOrderID); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el estado general de la comanda.")
		}
		if _, err = a.repo.SaveQRAppendRequest(tx.Context(), s, tableID, in, openOrderID); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la idempotencia del pedido.")
		}
		if _, err = a.repo.AuditQRAppend(tx.Context(), s, tableID, in, openOrderID, openChannel); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la trazabilidad del pedido.")
		}
		if err = tx.Commit(ctx); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar los productos adicionales.")
		}

		var out domain.Order
		out, err = a.repo.Record(ctx, s, openOrderID)
		if err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar la comanda actualizada.")
		}
		out.Items, err = a.repo.Items(ctx, s, out.ID, "")
		if err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar los productos actualizados.")
		}
		for _, item := range out.Items {
			if qty, parseErr := strconv.ParseFloat(item.Qty, 64); parseErr == nil {
				out.ItemCount += int(qty)
			}
		}
		_ = a.paymentSummary(ctx, s, &out)
		return TableSubmission{Order: out}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar las comandas abiertas de la mesa.")
	}

	prepared, subtotal, preparationErr := a.prepareItems(tx.Context(), s, "", in.Items)
	if preparationErr != nil {
		return result, preparationErr
	}

	var out domain.Order
	out, err = a.repo.CreateQROrder(tx.Context(), s, tableID, in, subtotal)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar el pedido.")
	}
	roundID, _, roundErr := a.repo.CreateKitchenRound(tx.Context(), s, out.ID, "confirmado", "concierge")
	if roundErr != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos abrir la primera ronda de cocina.")
	}
	if err = a.repo.InsertItems(tx.Context(), s, out.ID, roundID, prepared); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar los productos.")
	}
	if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, out.ID, domain.PreparedQuantityUsage(prepared), "sale"); quantityErr != nil {
		return result, quantityErr
	}
	recipeUsage, recipeErr := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(prepared))
	if recipeErr != nil {
		return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el consumo de recetas.")
	}
	if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, out.ID, recipeUsage); quantityErr != nil {
		return result, quantityErr
	}
	if _, err = a.repo.SaveQRCreateRequest(tx.Context(), s, tableID, in, out); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la idempotencia del pedido.")
	}
	if _, err = a.repo.AuditQRCreated(tx.Context(), s, tableID, in, out); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la trazabilidad del pedido.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar el pedido.")
	}

	out.Items, _ = a.repo.Items(ctx, s, out.ID, "")
	for _, item := range out.Items {
		if qty, parseErr := strconv.ParseFloat(item.Qty, 64); parseErr == nil {
			out.ItemCount += int(qty)
		}
	}
	_ = a.paymentSummary(ctx, s, &out)
	return TableSubmission{Order: out, Created: true}, nil

}

func (a *Orders) TableBill(ctx context.Context, s scope, tableID string) (domain.Order, error) {
	o, err := a.repo.OpenTableOrder(ctx, s, tableID)
	if errors.Is(err, ErrNotFound) {
		return o, orderFailure(apperror.NotFound, "open_order_not_found", "La mesa no tiene una cuenta abierta.")
	}
	if err != nil {
		return o, orderFailure(apperror.Unavailable, "bill_unavailable", "No pudimos cargar la cuenta de la mesa.")
	}
	o.Items, err = a.repo.Items(ctx, s, o.ID, "")
	if err != nil {
		return o, orderFailure(apperror.Unavailable, "bill_unavailable", "No pudimos cargar el detalle de la cuenta.")
	}
	if err = a.paymentSummary(ctx, s, &o); err != nil {
		return o, orderFailure(apperror.Unavailable, "bill_unavailable", "No pudimos calcular el saldo pendiente.")
	}
	return o, nil
}
