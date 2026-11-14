package application

import (
	"context"
	"errors"
	deliverycontracts "github.com/foods-platform/foods-backend/internal/modules/delivery/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ConciergeDeliveryInput = contracts.ConciergeDeliveryInput
type ConciergeOrderState = contracts.ConciergeOrderState

// ConciergeDeliveryReplay devuelve el pedido ya creado para requestId, si existe.
func (a *Orders) ConciergeDeliveryReplay(ctx context.Context, s scope, requestID string) (domain.Order, bool, error) {
	id, err := a.repo.ConciergeDeliveryRequestOrder(ctx, s, strings.TrimSpace(requestID))
	if errors.Is(err, ErrNotFound) {
		return domain.Order{}, false, nil
	}
	if err != nil {
		return domain.Order{}, false, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la idempotencia del pedido.")
	}
	out, err := a.conciergeOrderView(ctx, s, id)
	return out, err == nil, err
}

// SubmitConciergeDelivery crea un pedido de delivery o recojo confirmado y
// enviado a cocina. requestId es idempotente: repetirlo devuelve el mismo pedido.
func (a *Orders) SubmitConciergeDelivery(ctx context.Context, s scope, in ConciergeDeliveryInput) (TableSubmission, error) {
	var result TableSubmission
	if message := normalizeConciergeDelivery(&in); message != "" {
		return result, orderFailure(apperror.Invalid, "invalid_order", message)
	}
	if in.OrderID != "" {
		return a.appendConciergeDelivery(ctx, s, in)
	}
	return a.submitWithoutTable(ctx, s, in, nil)
}

// SubmitCounter registra una venta por mostrador desde su QR: un pedido
// independiente, sin mesa, cuenta abierta ni mozo, confirmado y enviado a
// preparación, que se cobra en caja. Varios clientes con el mismo QR crean
// pedidos distintos y el QR nunca se bloquea por pedidos abiertos; requestId
// evita duplicar el mismo pedido.
func (a *Orders) SubmitCounter(ctx context.Context, s scope, counterTableID string, token string, in QROrderInput) (TableSubmission, error) {
	order := ConciergeDeliveryInput{
		Channel:        domain.ChannelCounter,
		CustomerName:   in.CustomerName,
		CustomerPhone:  in.CustomerPhone,
		ConversationID: in.ConversationID,
		RequestID:      in.RequestID,
		Notes:          in.Notes,
		Items:          in.Items,
		CounterTableID: counterTableID,
	}
	if message := normalizeCounterOrder(&order); message != "" {
		return TableSubmission{}, orderFailure(apperror.Invalid, "invalid_order", message)
	}
	// El QR sigue activo y es de mostrador: FOR SHARE permite pedidos
	// simultáneos y espera solo a un cambio de tipo en curso.
	validate := func(txCtx context.Context) error {
		err := a.repo.ShareCounterQR(txCtx, s, counterTableID, token)
		if errors.Is(err, ErrNotFound) {
			return orderFailure(apperror.NotFound, "table_not_found", "El mostrador no existe o el QR ya no está activo.")
		}
		if err != nil {
			return orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar el mostrador.")
		}
		return nil
	}
	return a.submitWithoutTable(ctx, s, order, validate)
}

// submitWithoutTable es el registro común de pedidos de Concierge que no
// ocupan mesa (delivery, recojo y mostrador): idempotencia por requestId,
// cliente prerregistrado, inventario y envío a preparación en una transacción.
func (a *Orders) submitWithoutTable(ctx context.Context, s scope, in ConciergeDeliveryInput, validate func(context.Context) error) (TableSubmission, error) {
	var result TableSubmission
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos iniciar el pedido.")
	}
	defer tx.Rollback(ctx)

	if err = a.repo.LockConciergeDeliveryRequest(tx.Context(), s, in.RequestID); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la idempotencia del pedido.")
	}
	existingID, err := a.repo.ConciergeDeliveryRequestOrder(tx.Context(), s, in.RequestID)
	if err == nil {
		if err = tx.Commit(ctx); err != nil {
			return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar el pedido confirmado.")
		}
		out, viewErr := a.conciergeOrderView(ctx, s, existingID)
		return TableSubmission{Order: out}, viewErr
	}
	if !errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos validar la idempotencia del pedido.")
	}
	if validate != nil {
		if err = validate(tx.Context()); err != nil {
			return result, err
		}
	}

	prepared, subtotal, preparationErr := a.prepareItems(tx.Context(), s, "", in.Items)
	if preparationErr != nil {
		return result, preparationErr
	}
	customerID, err := a.repo.RegisterConciergeCustomer(tx.Context(), s, in.CustomerPhone, in.CustomerName)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar al cliente del pedido.")
	}
	created, err := a.repo.CreateConciergeDeliveryOrder(tx.Context(), s, in, subtotal, subtotal+in.DeliveryFee, customerID)
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar el pedido.")
	}
	if err = a.repo.InsertItems(tx.Context(), s, created.ID, "", prepared); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos guardar los productos del pedido.")
	}
	if quantityErr := a.stock.ApplyOrderQuantityDelta(tx.Context(), s, created.ID, domain.PreparedQuantityUsage(prepared), "sale"); quantityErr != nil {
		return result, quantityErr
	}
	recipeUsage, recipeErr := a.stock.DesiredRecipeUsage(tx.Context(), s, domain.PreparedRecipeProductUsage(prepared))
	if recipeErr != nil {
		return result, orderFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el consumo de recetas.")
	}
	if quantityErr := a.stock.ApplyRecipeUsageDelta(tx.Context(), s, created.ID, recipeUsage); quantityErr != nil {
		return result, quantityErr
	}
	// Igual que un pedido manual enviado a cocina: los productos quedan
	// pendientes de preparación por destino (Cocina/Barra/Sin preparación).
	if err = a.repo.SyncServiceStatus(tx.Context(), s, created.ID); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos enviar el pedido a cocina.")
	}
	if err = a.repo.SaveConciergeDeliveryRequest(tx.Context(), s, in, created.ID); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la idempotencia del pedido.")
	}
	if err = a.repo.AuditConciergeDeliveryCreated(tx.Context(), s, in, created.ID); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos registrar la trazabilidad del pedido.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos confirmar el pedido.")
	}
	out, err := a.conciergeOrderView(ctx, s, created.ID)
	return TableSubmission{Order: out, Created: true}, err
}

// ConciergeOrder lee un pedido solo si pertenece a la conversación indicada.
func (a *Orders) ConciergeOrder(ctx context.Context, s scope, conversationID, orderID string) (ConciergeOrderState, error) {
	var result ConciergeOrderState
	conversationID = strings.TrimSpace(conversationID)
	orderID = strings.TrimSpace(orderID)
	if conversationID == "" || len(conversationID) > 160 || orderID == "" {
		return result, orderFailure(apperror.Invalid, "invalid_order", "La conversación o el pedido no son válidos.")
	}
	active, err := a.repo.ConciergeOrderActive(ctx, s, conversationID, orderID)
	if errors.Is(err, ErrNotFound) {
		return result, orderFailure(apperror.NotFound, "order_not_found", "El pedido no pertenece a esta conversación.")
	}
	if err != nil {
		return result, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos consultar el pedido.")
	}
	out, err := a.conciergeOrderView(ctx, s, orderID)
	if err != nil {
		return result, err
	}
	canRequestBill, err := a.canRequestTableBill(ctx, s, out)
	if err != nil {
		return result, err
	}
	return ConciergeOrderState{Order: out, Active: active, CanRequestBill: active && canRequestBill}, nil
}

func (a *Orders) conciergeOrderView(ctx context.Context, s scope, id string) (domain.Order, error) {
	out, err := a.repo.Record(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return out, orderFailure(apperror.NotFound, "order_not_found", "El pedido no existe en este local.")
	}
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar el pedido.")
	}
	out.Items, err = a.repo.Items(ctx, s, out.ID, "")
	if err != nil {
		return out, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar los productos del pedido.")
	}
	for _, item := range out.Items {
		if qty, parseErr := strconv.ParseFloat(item.Qty, 64); parseErr == nil {
			out.ItemCount += int(qty)
		}
	}
	if err = a.paymentSummary(ctx, s, &out); err != nil {
		return out, orderFailure(apperror.Unavailable, "order_unavailable", "No pudimos recuperar el estado del pedido.")
	}
	return out, nil
}

func normalizeConciergeDelivery(in *ConciergeDeliveryInput) string {
	in.Channel = strings.TrimSpace(in.Channel)
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.RequestID = strings.TrimSpace(in.RequestID)
	in.Address = strings.TrimSpace(in.Address)
	in.Reference = strings.TrimSpace(in.Reference)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.Channel != "delivery" && in.Channel != "recojo" {
		return "El pedido debe ser delivery o recojo en local."
	}
	if message := validateOrderWithoutTable(in); message != "" {
		return message
	}
	if in.OrderID != "" {
		if matched, _ := regexp.MatchString(`^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$`, in.OrderID); !matched {
			return "El pedido de origen no es válido."
		}
		quoted, err := strconv.ParseFloat(in.ExpectedOrderTotal, 64)
		if err != nil || math.IsNaN(quoted) || math.IsInf(quoted, 0) || quoted < 0 {
			return "Revisa el total del pedido antes de agregar productos."
		}
		return ""
	}
	if in.Channel == "recojo" {
		if len(in.CustomerPhone) > 32 {
			return "Ingresa un teléfono válido de 7 a 15 dígitos."
		}
		in.Address, in.Reference, in.DeliveryFee = "", "", 0
		in.Latitude, in.Longitude, in.DistanceKm = nil, nil, nil
		return ""
	}
	if message := deliverycontracts.ValidateContact(in.Channel, in.CustomerName, in.CustomerPhone, in.Address, in.Reference); message != "" {
		return message
	}
	if in.Latitude == nil || in.Longitude == nil || !deliverycontracts.ValidCoordinates(*in.Latitude, *in.Longitude) {
		return "Comparte la ubicación de entrega."
	}
	if in.DeliveryFee < 0 || in.DistanceKm == nil || *in.DistanceKm < 0 {
		return "No pudimos calcular el costo de envío."
	}
	return ""
}

// normalizeCounterOrder: el cliente recoge en el mostrador, sin dirección ni envío.
func normalizeCounterOrder(in *ConciergeDeliveryInput) string {
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.ConversationID = strings.TrimSpace(in.ConversationID)
	in.RequestID = strings.TrimSpace(in.RequestID)
	in.Notes = strings.TrimSpace(in.Notes)
	in.Address, in.Reference, in.DeliveryFee = "", "", 0
	in.Latitude, in.Longitude, in.DistanceKm = nil, nil, nil
	if in.CounterTableID == "" {
		return "Revisa los productos y datos del pedido."
	}
	if message := validateOrderWithoutTable(in); message != "" {
		return message
	}
	if len(in.CustomerPhone) > 32 {
		return "Ingresa un teléfono válido de 7 a 15 dígitos."
	}
	return ""
}

func validateOrderWithoutTable(in *ConciergeDeliveryInput) string {
	if len(in.Items) == 0 || len(in.Items) > 50 || in.ConversationID == "" || len(in.ConversationID) > 160 ||
		in.RequestID == "" || len(in.RequestID) > 160 || utf8.RuneCountInString(in.Notes) > 240 {
		return "Revisa los productos y datos del pedido."
	}
	if in.CustomerName == "" || utf8.RuneCountInString(in.CustomerName) > 160 {
		return "Ingresa el nombre del cliente (máximo 160 caracteres)."
	}
	return ""
}
