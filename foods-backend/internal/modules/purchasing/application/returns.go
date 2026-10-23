package application

import (
	"context"
	"errors"
	inventorycontracts "github.com/foods-platform/foods-backend/internal/modules/inventory/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strings"
	"unicode/utf8"
)

func (a *Purchasing) Return(ctx context.Context, s scope, id string, in domain.PurchaseReturnInput) (map[string]any, error) {
	var result map[string]any
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Kind = strings.TrimSpace(in.Kind)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Notes = strings.TrimSpace(in.Notes)
	if len(in.IdempotencyKey) > 120 || (in.Kind != "supplier_return" && in.Kind != "receipt_correction") || in.Reason == "" || utf8.RuneCountInString(in.Reason) > 160 || utf8.RuneCountInString(in.Notes) > 500 || len(in.Items) == 0 {
		return result, purchaseFailure(apperror.Invalid, "invalid_purchase_return", "Indica tipo, motivo y cantidades a devolver.")
	}
	seen := map[string]bool{}
	for i := range in.Items {
		in.Items[i].PurchaseReceiptItemID = strings.TrimSpace(in.Items[i].PurchaseReceiptItemID)
		in.Items[i].Quantity = math.Round(in.Items[i].Quantity*1000) / 1000
		if in.Items[i].PurchaseReceiptItemID == "" || in.Items[i].Quantity <= 0 || seen[in.Items[i].PurchaseReceiptItemID] {
			return result, purchaseFailure(apperror.Invalid, "invalid_purchase_return", "Las líneas de devolución no son válidas.")
		}
		seen[in.Items[i].PurchaseReceiptItemID] = true
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos iniciar la devolución.")
	}
	defer tx.Rollback(ctx)
	var receiptID, orderID, supplierID, number string
	err = func() error {
		var err error
		receiptID, orderID, supplierID, number, err = a.repo.LockReturnReceipt(tx.Context(), s, id)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, purchaseFailure(apperror.NotFound, "purchase_receipt_not_found", "La recepción no existe en este local.")
	}
	if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos validar la recepción.")
	}
	if in.IdempotencyKey != "" {
		var existingID, existingCode, existingKind string
		err = func() error {
			var err error
			existingID, existingCode, existingKind, err = a.repo.ExistingReturn(tx.Context(), s, receiptID, in)
			return err
		}()
		if err == nil {
			return map[string]any{"id": existingID, "code": existingCode, "kind": existingKind, "purchaseOrderId": orderID, "number": number, "idempotent": true}, nil
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos validar la devolución previa.")
		}
	}

	var returnID, code string
	err = func() error {
		var err error
		returnID, code, err = a.repo.CreateReturn(tx.Context(), s, supplierID, orderID, receiptID, in)
		return err
	}()
	if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos crear la devolución.")
	}

	for _, requested := range in.Items {
		var receiptItemID, orderItemID, itemID, presentationType string
		var receivedQty, units, unitCost, alreadyReturned float64
		err = func() error {
			var err error
			receiptItemID, orderItemID, itemID, receivedQty, presentationType, units, unitCost, alreadyReturned, err = a.repo.LockReturnLine(tx.Context(), s, receiptID, requested)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return result, purchaseFailure(apperror.Conflict, "purchase_return_item_invalid", "Una línea no pertenece a esta recepción.")
		}
		if err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos validar una línea.")
		}
		if requested.Quantity > receivedQty-alreadyReturned+0.000001 {
			return result, purchaseFailure(apperror.Conflict, "purchase_return_exceeds_received", "La devolución supera lo recibido disponible.")
		}
		if presentationType != "unit" && math.Abs(requested.Quantity-math.Round(requested.Quantity)) > 0.000001 {
			return result, purchaseFailure(apperror.Invalid, "invalid_purchase_return_quantity", "Esta presentación se devuelve en cantidades enteras; para fracciones usa la unidad de inventario.")
		}
		if err = a.stock.ReturnPurchase(tx.Context(), s, inventorycontracts.PurchaseReturn{
			ItemID: itemID, Quantity: requested.Quantity, UnitsPerPresentation: units, UnitCost: unitCost,
			Kind: in.Kind, ReturnID: returnID, Note: in.Reason,
		}); errors.Is(err, inventorycontracts.ErrInsufficientStock) {
			return result, purchaseFailure(apperror.Conflict, "insufficient_stock", "No hay stock suficiente para registrar la devolución.")
		} else if err != nil {
			return result, purchaseFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos registrar la devolución en inventario.")
		}
		if _, err = a.repo.CreateReturnItem(tx.Context(), s, returnID, receiptItemID, orderItemID, itemID, requested, presentationType, units, unitCost); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos guardar el detalle de devolución.")
		}
		if in.Kind == "receipt_correction" {
			if _, err = a.repo.CorrectReceivedQuantity(tx.Context(), s, orderItemID, orderID, requested); err != nil {
				return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos corregir la recepción.")
			}
		}
	}
	if in.Kind == "receipt_correction" {
		var receivedLines, pendingLines int
		if err = func() error {
			var err error
			receivedLines, pendingLines, err = a.repo.ReturnOrderCounts(tx.Context(), s, orderID)
			return err
		}(); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos recalcular la orden.")
		}
		next := "approved"
		if receivedLines > 0 {
			next = "partially_received"
		}
		if pendingLines == 0 {
			next = "received"
		}
		if _, err = a.repo.ReturnOrderStatus(tx.Context(), s, orderID, next); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos actualizar la orden.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_return_unavailable", "No pudimos confirmar la devolución.")
	}
	_ = a.audit.Record(ctx, s, "purchase.return_created", "purchase_order", orderID)
	return map[string]any{"id": returnID, "code": code, "kind": in.Kind, "purchaseOrderId": orderID, "number": number}, nil

}
func (a *Purchasing) Approve(ctx context.Context, s scope, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos aprobar la orden.")
	}
	defer tx.Rollback(ctx)
	var status string
	err = func() error { var err error; status, err = a.repo.LockApprovalOrder(tx.Context(), s, id); return err }()
	if errors.Is(err, ErrNotFound) {
		return purchaseFailure(apperror.NotFound, "purchase_not_found", "La orden de compra no existe.")
	}
	if err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar la orden.")
	}
	if status != "pending_approval" {
		return purchaseFailure(apperror.Conflict, "purchase_transition_invalid", "Solo una orden por aprobar puede aprobarse.")
	}
	if _, err = a.repo.ApproveOrder(tx.Context(), s, id); err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos aprobar la orden.")
	}
	if err = tx.Commit(ctx); err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos confirmar la aprobación.")
	}
	_ = a.audit.Record(ctx, s, "purchase.approved", "purchase_order", id)
	return nil

}
