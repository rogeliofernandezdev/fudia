package application

import (
	"context"
	"errors"
	deliverydomain "github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strconv"
	"strings"
)

func (a *Orders) Create(ctx context.Context, s scope, in domain.OrderInput) (domain.Order, error) {
	var result domain.Order
	in.Channel = strings.TrimSpace(in.Channel)
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if !domain.ValidOrderChannel(in.Channel) || len(in.Items) == 0 || in.DeliveryFee < 0 {
		return result, orderFailure(apperror.Invalid, "invalid_order", "El canal y al menos un producto son obligatorios.")
	}
	if message := deliverydomain.ValidateContact(in.Channel, in.CustomerName, in.CustomerPhone, in.Address, in.Reference); message != "" {
		return result, orderFailure(apperror.Invalid, "invalid_order", message)
	}
	if in.Channel == "salon" && strings.TrimSpace(in.TableID) == "" {
		return result, orderFailure(apperror.Invalid, "table_required", "Los pedidos de Salón deben estar asociados a una mesa.")
	}
	if in.Channel != "salon" && strings.TrimSpace(in.TableID) != "" {
		return result, orderFailure(apperror.Invalid, "table_not_allowed", "Solo los pedidos de Salón pueden ocupar una mesa.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar el pedido.")
	}
	defer tx.Rollback(ctx)

	var customerID, tableID *string
	if in.CustomerID != "" {
		var ok bool
		if err = func() error { var err error; ok, err = a.repo.CustomerActive(tx.Context(), s, in); return err }(); err != nil || !ok {
			return result, orderFailure(apperror.Invalid, "invalid_order", "El cliente no existe.")
		}
		customerID = &in.CustomerID
	}
	if in.TableID != "" {
		var lockedTableID string
		_ = lockedTableID
		if err = func() error { var err error; lockedTableID, err = a.repo.LockTable(tx.Context(), s, in); return err }(); errors.Is(err, ErrNotFound) {
			return result, orderFailure(apperror.Invalid, "invalid_order", "La mesa no existe.")
		} else if err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos bloquear la mesa para abrir la comanda.")
		}
		var occupied bool
		if err = func() error { var err error; occupied, err = a.repo.TableOccupied(tx.Context(), s, in); return err }(); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar el pedido.")
		}
		if occupied {
			return result, orderFailure(apperror.Conflict, "table_occupied", "La mesa ya tiene un pedido abierto.")
		}
		tableID = &in.TableID
	}

	prepared, subtotal, preparationErr := a.prepareItems(tx.Context(), s, "", in.Items)
	if preparationErr != nil {
		return result, preparationErr
	}
	total := subtotal + in.DeliveryFee
	initialStatus := "nuevo"
	if in.SendToKitchen {
		initialStatus = "confirmado"
	}

	var o domain.Order
	o, err = a.repo.CreateOrder(tx.Context(), s, in, customerID, tableID, subtotal, total, initialStatus)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar el pedido.")
	}
	if err = a.repo.InsertItems(tx.Context(), s, o.ID, "", prepared); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar los productos del pedido.")
	}
	if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, o.ID, domain.PreparedQuantityUsage(prepared), "sale"); quantityErr != nil {
		return result, quantityErr
	}
	recipeUsage, recipeErr := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(prepared))
	if recipeErr != nil {
		return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el consumo de recetas.")
	}
	if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, o.ID, recipeUsage); quantityErr != nil {
		return result, quantityErr
	}
	if in.SendToKitchen {
		if err = a.repo.SyncServiceStatus(tx.Context(), s, o.ID); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar la preparación.")
		}
		o, err = a.repo.Record(tx.Context(), s, o.ID)
		if err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos leer el pedido.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar el pedido.")
	}
	_ = a.audit.Record(ctx, s, "created", "order", o.ID)
	if in.SendToKitchen {
		_ = a.audit.Record(ctx, s, "kitchen.submitted", "order", o.ID)
	}
	_ = a.paymentSummary(ctx, s, &o)
	return o, nil

}
func (a *Orders) Update(ctx context.Context, s scope, id string, in domain.OrderUpdateInput) (domain.Order, error) {
	var result domain.Order
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	if len(in.Items) == 0 || in.DeliveryFee < 0 {
		return result, orderFailure(apperror.Invalid, "invalid_order", "La comanda necesita al menos un producto y montos válidos.")
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	defer tx.Rollback(ctx)

	var currentStatus, channel, waiterID string
	err = func() error {
		var err error
		currentStatus, channel, waiterID, err = a.repo.LockEditableOrder(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	if !domain.CanManageOrderService(s.UserID, channel, waiterID) {
		return result, orderFailure(apperror.Forbidden, "order_assigned_to_another_waiter", "Esta mesa está siendo atendida por otro mozo. Solo puedes consultar su detalle.")
	}
	if !domain.EditableOrderStatus(currentStatus) {
		return result, orderFailure(apperror.Conflict, "order_not_editable", "Solo se puede editar un pedido en estado Nuevo o Confirmado.")
	}
	if channel == "salon" && currentStatus != "nuevo" {
		var sent bool
		if err = func() error { var err error; sent, err = a.repo.HasServiceItems(tx.Context(), s, id); return err }(); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la comanda.")
		}
		if sent {
			return result, orderFailure(apperror.Conflict, "sent_order_not_editable", "Agrega productos a la cuenta sin reemplazar la comanda enviada.")
		}
	}
	var hasKitchenRounds bool
	if err = func() error {
		var err error
		hasKitchenRounds, err = a.repo.HasKitchenRounds(tx.Context(), s, id)
		return err
	}(); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar las rondas de cocina.")
	}
	if hasKitchenRounds {
		return result, orderFailure(apperror.Conflict, "round_order_not_editable", "Esta comanda ya tiene rondas enviadas a cocina. Agrega una nueva ronda en lugar de reemplazar sus productos.")
	}
	paid, paidErr := a.repo.NetPaid(tx.Context(), s, id)
	if paidErr != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar los pagos del pedido.")
	}
	if paid > 0.00001 {
		return result, orderFailure(apperror.Conflict, "paid_order_not_editable", "No se puede modificar una comanda después de registrar pagos. Devuelve los pagos antes de editarla.")
	}
	if channel == "delivery" && strings.TrimSpace(in.Address) == "" {
		return result, orderFailure(apperror.Invalid, "invalid_order", "El pedido de delivery necesita una dirección.")
	}

	previousUsage, err := a.repo.QuantityUsage(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar las cantidades actuales del pedido.")
	}
	previousRecipeUsage, err := a.stock.NetRecipeUsage(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos cargar el consumo actual de recetas.")
	}
	prepared, subtotal, preparationErr := a.prepareItems(tx.Context(), s, id, in.Items)
	if preparationErr != nil {
		return result, preparationErr
	}
	if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, id, domain.QuantityUsageDelta(previousUsage, domain.PreparedQuantityUsage(prepared)), "sale_adjustment"); quantityErr != nil {
		return result, quantityErr
	}
	nextRecipeUsage, recipeErr := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(prepared))
	if recipeErr != nil {
		return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el consumo de recetas.")
	}
	if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, id, domain.QuantityUsageDelta(previousRecipeUsage, nextRecipeUsage)); quantityErr != nil {
		return result, quantityErr
	}
	total := subtotal + in.DeliveryFee

	_, err = a.repo.UpdateOrder(tx.Context(), s, id, in, subtotal, total)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	if _, err = a.repo.DeleteItems(tx.Context(), s, id); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar los productos del pedido.")
	}
	if err = a.repo.InsertItems(tx.Context(), s, id, "", prepared); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar los productos del pedido.")
	}

	o, err := a.repo.Record(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el pedido actualizado.")
	}
	o.Items, err = a.repo.Items(tx.Context(), s, o.ID, "")
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar los productos actualizados.")
	}
	for _, it := range o.Items {
		if qty, parseErr := strconv.ParseFloat(it.Qty, 64); parseErr == nil {
			o.ItemCount += int(qty)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	_ = a.audit.Record(ctx, s, "updated", "order", o.ID)
	return o, nil

}
func (a *Orders) SetStatus(ctx context.Context, s scope, id string, in OrderStatusInput) (domain.Order, error) {
	var result domain.Order
	if !domain.ValidOrderStatus(in.Status) {
		return result, orderFailure(apperror.Invalid, "invalid_status", "El estado no es válido.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	defer tx.Rollback(ctx)

	var current, channel, tableID, waiterID string
	var completed bool
	var total float64
	err = func() error {
		var err error
		current, channel, tableID, total, completed, waiterID, err = a.repo.LockOrderStatus(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	if !domain.CanManageOrderService(s.UserID, channel, waiterID) {
		return result, orderFailure(apperror.Forbidden, "order_assigned_to_another_waiter", "Esta mesa está siendo atendida por otro mozo. Solo puedes consultar su detalle.")
	}
	if (current == "confirmado" && in.Status == "preparando") || (current == "preparando" && in.Status == "listo") {
		return result, orderFailure(apperror.Conflict, "kitchen_transition_required", "Los estados de preparación solo se actualizan desde Cocina.")
	}
	var hasService bool
	if in.Status == "cancelado" {
		var started bool
		if err = func() error {
			var err error
			hasService, started, err = a.repo.ServiceStarted(tx.Context(), s, id)
			return err
		}(); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar los productos.")
		}
		if started || (!hasService && (current == "preparando" || current == "listo" || current == "en_camino")) {
			return result, orderFailure(apperror.Conflict, "cancellation_requires_void", "Ya hay productos preparados o entregados. Esta cuenta requiere un flujo de anulación.")
		}
	}
	releasingTable := current == "entregado" && in.Status == "entregado" && channel == "salon" && tableID != "" && !completed
	allowed := releasingTable
	for _, next := range domain.Transitions[current] {
		if next == in.Status {
			allowed = true
			break
		}
	}
	if in.Status == "cancelado" {
		if hasService {
			// Direct delivery is ready immediately, not prepared. Its unserved
			// quantities can still be reversed if no station has started work.
			allowed = current == "nuevo" || current == "confirmado" || current == "preparando" || current == "listo"
		}
		allowed = allowed && !completed
	}
	if !allowed {
		return result, orderFailure(apperror.Conflict, "invalid_transition", "No se puede pasar de "+current+" a "+in.Status+".")
	}

	paid, paidErr := a.repo.NetPaid(tx.Context(), s, id)
	if paidErr != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el estado de cobro del pedido.")
	}
	if in.Status == "entregado" && tableID != "" && (channel != "salon" || releasingTable) && paid+0.00001 < total {
		return result, orderFailure(apperror.Conflict, "payment_required_before_delivery", "La mesa no puede liberarse mientras exista saldo pendiente.")
	}
	if in.Status == "cancelado" && paid > 0.00001 {
		return result, orderFailure(apperror.Conflict, "refund_required_before_cancel", "Devuelve primero los pagos registrados antes de cancelar el pedido.")
	}

	if in.Status == "cancelado" {
		usage, usageErr := a.repo.QuantityUsage(tx.Context(), s, id)
		if usageErr != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar las cantidades del pedido.")
		}
		if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, id, domain.QuantityUsageDelta(usage, map[string]float64{}), "sale_reversal"); quantityErr != nil {
			return result, quantityErr
		}
		recipeUsage, recipeErr := a.stock.NetRecipeUsage(tx.Context(), s, id)
		if recipeErr != nil {
			return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos cargar el consumo de recetas del pedido.")
		}
		if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, id, domain.QuantityUsageDelta(recipeUsage, map[string]float64{})); quantityErr != nil {
			return result, quantityErr
		}
	}

	if in.Status == "confirmado" {
		if _, err = a.repo.SubmitServiceItems(tx.Context(), s, id); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar los productos.")
		}
	}
	if in.Status == "entregado" {
		var pending bool
		if err = func() error {
			var err error
			pending, err = a.repo.PendingServiceItems(tx.Context(), s, id)
			return err
		}(); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la entrega.")
		}
		if pending {
			return result, orderFailure(apperror.Conflict, "items_not_ready", "Todavía hay productos en preparación.")
		}
		if _, err = a.repo.DeliverServiceItems(tx.Context(), s, id); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la entrega.")
		}
	}
	tag, err := a.repo.SetStatus(tx.Context(), s, id, in)
	if err != nil || tag == 0 {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe.")
	}
	if in.Status == "confirmado" {
		if err = a.repo.SyncServiceStatus(tx.Context(), s, id); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar la atención.")
		}
	}
	if _, err = a.repo.CompleteDeliveredAccount(tx.Context(), s, id); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos finalizar la cuenta del pedido.")
	}
	o, err := a.repo.Record(tx.Context(), s, id)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el pedido actualizado.")
	}
	if err = a.paymentSummary(tx.Context(), s, &o); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos cargar el estado de cobro del pedido.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos actualizar el pedido.")
	}
	action := "status_updated"
	if in.Status == "entregado" {
		action = "order.delivered"
		if o.CompletedAt != "" {
			action = "order.completed"
		}
	}
	_ = a.audit.Record(ctx, s, action, "order", id)
	return o, nil

}

type OrderStatusInput struct {
	Status string `json:"status"`
}
type OrderStock interface {
	ApplyOrderQuantityDelta(context.Context, scope, string, map[string]float64, string) error
	DesiredRecipeUsage(context.Context, scope, map[string]float64) (map[string]float64, error)
	NetRecipeUsage(context.Context, scope, string) (map[string]float64, error)
	ApplyRecipeUsageDelta(context.Context, scope, string, map[string]float64) error
}
