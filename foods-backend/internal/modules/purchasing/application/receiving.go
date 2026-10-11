package application

import (
	"context"
	"errors"
	inventoryapp "github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strings"
	"time"
)

func (a *Purchasing) Receive(ctx context.Context, s scope, id string, in domain.PurchaseReceiptInput) (map[string]any, error) {
	var result map[string]any
	in.Notes = strings.TrimSpace(in.Notes)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if len(in.IdempotencyKey) > 120 {
		return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt", "La clave de idempotencia es demasiado larga.")
	}
	if len(in.Notes) > 500 {
		return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt", "Las notas no pueden superar 500 caracteres.")
	}
	if len(in.Items) == 0 {
		return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt", "Registra al menos una cantidad recibida.")
	}
	if len(in.Items) > 100 {
		return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt", "La recepción no puede superar 100 líneas.")
	}
	seen := map[string]bool{}
	for index := range in.Items {
		line := &in.Items[index]
		line.PurchaseOrderItemID = strings.TrimSpace(line.PurchaseOrderItemID)
		line.Quantity = math.Round(line.Quantity*1000) / 1000
		if line.PurchaseOrderItemID == "" || line.Quantity <= 0 {
			return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt", "Cada línea recibida necesita artículo y cantidad mayor que cero.")
		}
		if seen[line.PurchaseOrderItemID] {
			return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt", "No repitas el mismo artículo en una recepción.")
		}
		seen[line.PurchaseOrderItemID] = true
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos recibir la orden.")
	}
	defer tx.Rollback(ctx)

	var number, status string
	if err = func() error {
		var err error
		number, status, err = a.repo.LockReceivingOrder(tx.Context(), s, id)
		return err
	}(); errors.Is(err, ErrNotFound) {
		return result, purchaseFailure(apperror.NotFound, "purchase_not_found", "La orden de compra no existe.")
	} else if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar la orden.")
	}
	if in.IdempotencyKey != "" {
		var existingID, existingCode string
		var existingStatus string
		err = func() error {
			var err error
			existingID, existingCode, existingStatus, err = a.repo.ExistingReceipt(tx.Context(), s, id, in)
			return err
		}()
		if err == nil {
			return map[string]any{"id": existingID, "code": existingCode, "purchaseOrderId": id, "number": number, "status": existingStatus, "idempotent": true}, nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar la recepción previa.")
		}
	}
	if status != "approved" && status != "partially_received" {
		return result, purchaseFailure(apperror.Conflict, "purchase_not_receivable", "Solo una orden aprobada o parcialmente recibida puede recibir mercadería.")
	}

	var receiptID, receiptCode string
	if err = func() error {
		var err error
		receiptID, receiptCode, err = a.repo.CreateReceipt(tx.Context(), s, id, in)
		return err
	}(); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos crear la recepción.")
	}

	lines := make([]domain.ReceivingLine, 0, len(in.Items))
	for _, requested := range in.Items {
		var line domain.ReceivingLine
		err = func() error {
			var err error
			line, err = a.repo.LockReceivingLine(tx.Context(), s, id, requested)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return result, purchaseFailure(apperror.Conflict, "purchase_receipt_item_invalid", "Una línea ya no pertenece a esta orden.")
		}
		if err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar los artículos de la recepción.")
		}
		line.ReceiveQuantity = requested.Quantity
		pending := math.Round((line.OrderedQuantity-line.ReceivedQuantity)*1000) / 1000
		if line.ReceiveQuantity-pending > 0.000001 {
			return result, purchaseFailure(apperror.Conflict, "purchase_receipt_exceeds_pending", "La cantidad recibida de "+line.ItemName+" supera lo pendiente.")
		}
		if line.PresentationType != "unit" && math.Abs(line.ReceiveQuantity-math.Round(line.ReceiveQuantity)) > 0.000001 {
			return result, purchaseFailure(apperror.Invalid, "invalid_purchase_receipt_quantity", "Paquetes y cajas deben recibirse en cantidades enteras.")
		}
		lines = append(lines, line)
	}

	for _, line := range lines {
		stockQuantity := math.Round(line.ReceiveQuantity*line.UnitsPerPresentation*1000) / 1000
		if _, err = a.repo.CreateReceiptItem(tx.Context(), s, receiptID, line); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos guardar el detalle de la recepción.")
		}
		if _, err = a.stock.EnsureEntryBalance(tx.Context(), s, line.InventoryItemID); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos preparar el saldo de inventario.")
		}
		var current, currentAverage float64
		if err = func() error {
			var err error
			current, currentAverage, err = a.stock.LockEntryBalance(tx.Context(), s, line.InventoryItemID)
			return err
		}(); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos bloquear el saldo de inventario.")
		}
		balanceAfter := math.Round((current+stockQuantity)*1000) / 1000
		baseUnitCost := 0.0
		if line.UnitsPerPresentation > 0 {
			baseUnitCost = line.UnitCost / line.UnitsPerPresentation
		}
		averageAfter := currentAverage
		if balanceAfter > 0 {
			averageAfter = math.Round(((current*currentAverage)+(stockQuantity*baseUnitCost))/balanceAfter*10000) / 10000
		}
		if _, err = a.stock.SaveBalance(tx.Context(), s, line.InventoryItemID, balanceAfter, averageAfter); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos actualizar el saldo de inventario.")
		}
		note := "Recepción " + receiptCode + " de " + number
		if in.Notes != "" {
			note += " · " + in.Notes
		}
		valueDelta := math.Round(stockQuantity*baseUnitCost*10000) / 10000
		balanceValue := math.Round(balanceAfter*averageAfter*10000) / 10000
		if _, err = func() (int64, error) {
			err := a.stock.SaveMovement(tx.Context(), s, inventoryapp.Movement{ProductID: line.ProductID, ItemID: line.InventoryItemID, Type: "entry", Delta: stockQuantity, Balance: balanceAfter, UnitCost: baseUnitCost, SourceType: "purchase_receipt", SourceID: receiptID, Note: note, ValueDelta: valueDelta, BalanceValue: balanceValue})
			return 0, err
		}(); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
		}
		if _, err = a.repo.ReceiveOrderLine(tx.Context(), s, id, line); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar lo recibido de la orden.")
		}
	}

	var pendingCount int
	if err = func() error { var err error; pendingCount, err = a.repo.PendingLines(tx.Context(), s, id); return err }(); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos calcular lo pendiente de la orden.")
	}
	nextStatus := "partially_received"
	var receivedAt *time.Time
	if pendingCount == 0 {
		nextStatus = "received"
		now := time.Now().UTC()
		receivedAt = &now
	}
	if _, err = a.repo.ReceiveOrderStatus(tx.Context(), s, id, nextStatus); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar la recepción de la orden.")
	}

	if err = tx.Commit(ctx); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos recibir la orden.")
	}
	_ = a.audit.Record(ctx, s, "purchase.receipt_created", "purchase_order", id)
	return map[string]any{
		"id":              receiptID,
		"code":            receiptCode,
		"purchaseOrderId": id,
		"number":          number,
		"status":          nextStatus,
		"receivedAt":      receivedAt,
	}, nil

}

type InventoryLedger interface {
	LockBalance(context.Context, scope, string) (inventorydomain.InventoryBalanceLock, error)
	EnsureEntryBalance(context.Context, scope, string) (int64, error)
	LockEntryBalance(context.Context, scope, string) (float64, float64, error)
	SaveBalance(context.Context, scope, string, float64, float64) (int64, error)
	SaveMovement(context.Context, scope, inventoryapp.Movement) error
}
