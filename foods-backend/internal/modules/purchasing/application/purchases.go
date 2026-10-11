package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strings"
)

type scope = requestscope.Scope

var ErrNotFound = errors.New("purchase record not found")

type Filter struct {
	Query, Status, From, To string
	Page, Size              int
}
type Purchasing struct {
	stock InventoryLedger
	repo  PurchaseRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func New(repo PurchaseRepository, work unitofwork.Manager, audit audit.Repository, stock InventoryLedger) *Purchasing {
	return &Purchasing{stock: stock, repo: repo, work: work, audit: audit}
}
func purchaseFailure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}

type PurchaseRepository interface {
	LockReturnReceipt(ctx context.Context, s scope, id string) (string, string, string, string, error)
	ExistingReturn(ctx context.Context, s scope, receiptID string, in domain.PurchaseReturnInput) (string, string, string, error)
	CreateReturn(ctx context.Context, s scope, supplierID string, orderID string, receiptID string, in domain.PurchaseReturnInput) (string, string, error)
	LockReturnLine(ctx context.Context, s scope, receiptID string, requested domain.ReturnLineInput) (string, string, string, float64, string, float64, float64, float64, error)
	CreateReturnItem(ctx context.Context, s scope, returnID string, receiptItemID string, orderItemID string, itemID string, requested domain.ReturnLineInput, presentationType string, units float64, unitCost float64) (int64, error)
	CorrectReceivedQuantity(ctx context.Context, s scope, orderItemID string, orderID string, requested domain.ReturnLineInput) (int64, error)
	ReturnOrderCounts(ctx context.Context, s scope, orderID string) (int, int, error)
	ReturnOrderStatus(ctx context.Context, s scope, orderID string, next string) (int64, error)
	LockApprovalOrder(ctx context.Context, s scope, id string) (string, error)
	ApproveOrder(ctx context.Context, s scope, id string) (int64, error)
	LockReceivingOrder(ctx context.Context, s scope, id string) (string, string, error)
	ExistingReceipt(ctx context.Context, s scope, id string, in domain.PurchaseReceiptInput) (string, string, string, error)
	CreateReceipt(ctx context.Context, s scope, id string, in domain.PurchaseReceiptInput) (string, string, error)
	LockReceivingLine(ctx context.Context, s scope, id string, requested domain.PurchaseReceiptLineInput) (domain.ReceivingLine, error)
	CreateReceiptItem(ctx context.Context, s scope, receiptID string, line domain.ReceivingLine) (int64, error)
	ReceiveOrderLine(ctx context.Context, s scope, id string, line domain.ReceivingLine) (int64, error)
	PendingLines(ctx context.Context, s scope, id string) (int, error)
	ReceiveOrderStatus(ctx context.Context, s scope, id string, nextStatus string) (int64, error)
	UpdateSupplier(ctx context.Context, s scope, id string, in domain.SupplierInput) (domain.SupplierView, error)
	CreateSupplier(ctx context.Context, s scope, in domain.SupplierInput) (domain.SupplierView, error)
	SupplierHasOpenOrders(ctx context.Context, s scope, id string) (bool, error)
	SetSupplierActive(ctx context.Context, s scope, id string, in struct{ Active bool }) (int64, error)
	SupplierActive(ctx context.Context, s scope, in domain.PurchaseOrderInput) (bool, error)
	LockOrder(ctx context.Context, s scope, id string) (string, string, error)
	UpdateOrder(ctx context.Context, s scope, id string, in domain.PurchaseOrderInput) (int64, error)
	DeleteOrderItems(ctx context.Context, s scope, id string) (int64, error)
	CreateOrder(ctx context.Context, s scope, in domain.PurchaseOrderInput) (string, string, error)
	OrderPresentation(ctx context.Context, s scope, line domain.PurchaseOrderItemInput) (string, float64, error)
	CreateOrderItem(ctx context.Context, s scope, id string, line domain.PurchaseOrderItemInput, presentationType string, unitsPerPresentation float64) (int64, error)
	RecalculateOrder(ctx context.Context, s scope, id string) (int64, error)
	LockOrderStatus(ctx context.Context, s scope, id string) (string, error)
	SetOrderStatus(ctx context.Context, s scope, id string, in OrderStatusInput) (int64, error)
	Suppliers(context.Context, scope, Filter) (SuppliersPage, error)
	Orders(context.Context, scope, Filter) (OrdersPage, error)
	Receipts(context.Context, scope, Filter) (ReceiptsPage, error)
	Order(context.Context, scope, string) (domain.PurchaseOrderDetail, error)
	Receipt(context.Context, scope, string) (domain.PurchaseReceiptDetail, error)
}

func (a *Purchasing) SaveSupplier(ctx context.Context, s scope, id string, in domain.SupplierInput, updating bool) (domain.SupplierView, error) {
	var result domain.SupplierView
	var invalid string
	in, invalid = domain.NormalizeSupplier(in)
	if invalid != "" {
		return result, purchaseFailure(apperror.Invalid, "invalid_supplier", invalid)
	}
	var item domain.SupplierView
	var err error
	if updating {
		err = func() error { var err error; item, err = a.repo.UpdateSupplier(ctx, s, id, in); return err }()
	} else {
		err = func() error { var err error; item, err = a.repo.CreateSupplier(ctx, s, in); return err }()
	}
	if errors.Is(err, ErrNotFound) {
		return result, purchaseFailure(apperror.NotFound, "supplier_not_found", "El proveedor no existe.")
	}
	if err != nil {
		return result, purchaseFailure(apperror.Conflict, "supplier_conflict", "No pudimos guardar el proveedor. Revisa el RUC y los datos ingresados.")
	}
	_ = a.audit.Record(ctx, s, map[bool]string{true: "supplier.updated", false: "supplier.created"}[updating], "supplier", item.ID)
	return item, nil

}
func (a *Purchasing) SetSupplierActive(ctx context.Context, s scope, id string, active bool) error {
	if !active {
		var open bool
		if err := func() error { var err error; open, err = a.repo.SupplierHasOpenOrders(ctx, s, id); return err }(); err != nil {
			return purchaseFailure(apperror.Unavailable, "supplier_unavailable", "No pudimos validar el proveedor.")
		}
		if open {
			return purchaseFailure(apperror.Conflict, "supplier_in_use", "El proveedor tiene órdenes abiertas y no puede desactivarse.")
		}
	}
	tag, err := a.repo.SetSupplierActive(ctx, s, id, struct{ Active bool }{active})
	if err != nil || tag == 0 {
		return purchaseFailure(apperror.NotFound, "supplier_not_found", "El proveedor no existe.")
	}
	_ = a.audit.Record(ctx, s, "supplier.status_updated", "supplier", id)
	return nil

}
func (a *Purchasing) SaveOrder(ctx context.Context, s scope, orderID string, in domain.PurchaseOrderInput, updating bool) (map[string]string, error) {
	var result map[string]string
	var invalid string
	in, invalid = domain.NormalizePurchaseOrder(in)
	if invalid != "" {
		return result, purchaseFailure(apperror.Invalid, "invalid_purchase", invalid)
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos guardar la orden.")
	}
	defer tx.Rollback(ctx)

	var supplierActive bool
	if err = func() error {
		var err error
		supplierActive, err = a.repo.SupplierActive(tx.Context(), s, in)
		return err
	}(); errors.Is(err, ErrNotFound) {
		return result, purchaseFailure(apperror.Invalid, "supplier_not_found", "El proveedor seleccionado no existe.")
	} else if err != nil {
		return result, purchaseFailure(apperror.Unavailable, "supplier_unavailable", "No pudimos validar el proveedor.")
	} else if !supplierActive {
		return result, purchaseFailure(apperror.Conflict, "supplier_inactive", "El proveedor seleccionado está inactivo.")
	}

	var id, number string
	if updating {
		var status string
		err = func() error {
			var err error
			number, status, err = a.repo.LockOrder(tx.Context(), s, orderID)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return result, purchaseFailure(apperror.NotFound, "purchase_not_found", "La orden de compra no existe.")
		}
		if err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar la orden.")
		}
		if status != "draft" {
			return result, purchaseFailure(apperror.Conflict, "purchase_locked", "Solo una orden en borrador puede editarse.")
		}
		id = orderID
		if _, err = a.repo.UpdateOrder(tx.Context(), s, id, in); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar la orden.")
		}
		if _, err = a.repo.DeleteOrderItems(tx.Context(), s, id); err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar las líneas de la orden.")
		}
	} else {
		err = func() error { var err error; id, number, err = a.repo.CreateOrder(tx.Context(), s, in); return err }()
		if err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos crear la orden.")
		}
	}

	for _, line := range in.Items {
		var presentationType string
		var unitsPerPresentation float64
		err = func() error {
			var err error
			presentationType, unitsPerPresentation, err = a.repo.OrderPresentation(tx.Context(), s, line)
			return err
		}()
		if errors.Is(err, ErrNotFound) {
			return result, purchaseFailure(apperror.Conflict, "purchase_item_invalid", "Un artículo o presentación de la orden ya no está disponible.")
		}
		if err != nil {
			return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar los artículos de la orden.")
		}
		if presentationType != "unit" && math.Abs(line.Quantity-math.Round(line.Quantity)) > 0.000001 {
			return result, purchaseFailure(apperror.Invalid, "invalid_purchase_quantity", "Paquetes y cajas deben comprarse en cantidades enteras.")
		}
		if _, err = a.repo.CreateOrderItem(tx.Context(), s, id, line, presentationType, unitsPerPresentation); err != nil {
			return result, purchaseFailure(apperror.Conflict, "purchase_item_conflict", "No pudimos agregar una de las líneas de la orden.")
		}
	}
	if _, err = a.repo.RecalculateOrder(tx.Context(), s, id); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos calcular el total de la orden.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos guardar la orden.")
	}
	_ = a.audit.Record(ctx, s, map[bool]string{true: "purchase.updated", false: "purchase.created"}[updating], "purchase_order", id)
	return map[string]string{"id": id, "number": number, "status": "draft"}, nil

}
func (a *Purchasing) SetOrderStatus(ctx context.Context, s scope, id string, in OrderStatusInput) error {
	in.Status = strings.TrimSpace(in.Status)
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar la orden.")
	}
	defer tx.Rollback(ctx)
	var current string
	if err = func() error { var err error; current, err = a.repo.LockOrderStatus(tx.Context(), s, id); return err }(); errors.Is(err, ErrNotFound) {
		return purchaseFailure(apperror.NotFound, "purchase_not_found", "La orden de compra no existe.")
	} else if err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos validar la orden.")
	}
	allowed := map[string]map[string]bool{
		"draft":            {"pending_approval": true, "cancelled": true},
		"pending_approval": {"draft": true, "cancelled": true},
		"approved":         {"cancelled": true},
	}
	if !allowed[current][in.Status] {
		return purchaseFailure(apperror.Conflict, "purchase_transition_invalid", "La orden no puede pasar al estado solicitado.")
	}
	_, err = a.repo.SetOrderStatus(tx.Context(), s, id, in)
	if err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar el estado de la orden.")
	}
	if err = tx.Commit(ctx); err != nil {
		return purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos actualizar el estado de la orden.")
	}
	_ = a.audit.Record(ctx, s, "purchase.status_updated", "purchase_order", id)
	return nil

}

type OrderStatusInput struct {
	Status string `json:"status"`
}
type SuppliersPage struct {
	Items    []domain.SupplierView `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

func (a *Purchasing) Suppliers(ctx context.Context, s scope, f Filter) (SuppliersPage, error) {
	out, err := a.repo.Suppliers(ctx, s, f)
	if err != nil {
		return out, purchaseFailure(apperror.Unavailable, "suppliers_unavailable", "No pudimos cargar los proveedores.")
	}
	return out, nil
}

type OrdersPage struct {
	Items    []domain.PurchaseOrderSummary `json:"items"`
	Total    int                           `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"pageSize"`
}

func (a *Purchasing) Orders(ctx context.Context, s scope, f Filter) (OrdersPage, error) {
	out, err := a.repo.Orders(ctx, s, f)
	if err != nil {
		return out, purchaseFailure(apperror.Unavailable, "purchases_unavailable", "No pudimos cargar las órdenes de compra.")
	}
	return out, nil
}

type ReceiptsPage struct {
	Items    []domain.PurchaseReceiptSummary `json:"items"`
	Total    int                             `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"pageSize"`
}

func (a *Purchasing) Receipts(ctx context.Context, s scope, f Filter) (ReceiptsPage, error) {
	if !inventorydomain.ValidDateFilters(strings.TrimSpace(f.From), strings.TrimSpace(f.To)) {
		return ReceiptsPage{}, purchaseFailure(apperror.Invalid, "invalid_receipt_filter", "Revisa el rango de fechas.")
	}
	out, err := a.repo.Receipts(ctx, s, f)
	if err != nil {
		return out, purchaseFailure(apperror.Unavailable, "purchase_receipts_unavailable", "No pudimos cargar las recepciones.")
	}
	return out, nil
}
func (a *Purchasing) Order(ctx context.Context, s scope, id string) (domain.PurchaseOrderDetail, error) {
	out, err := a.repo.Order(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return out, purchaseFailure(apperror.NotFound, "purchase_not_found", "La orden de compra no existe.")
	}
	if err != nil {
		return out, purchaseFailure(apperror.Unavailable, "purchase_unavailable", "No pudimos cargar el detalle.")
	}
	return out, nil
}
func (a *Purchasing) Receipt(ctx context.Context, s scope, id string) (domain.PurchaseReceiptDetail, error) {
	out, err := a.repo.Receipt(ctx, s, id)
	if errors.Is(err, ErrNotFound) {
		return out, purchaseFailure(apperror.NotFound, "purchase_receipt_not_found", "La recepción no existe en este local.")
	}
	if err != nil {
		return out, purchaseFailure(apperror.Unavailable, "purchase_receipt_unavailable", "No pudimos cargar el detalle.")
	}
	return out, nil
}
