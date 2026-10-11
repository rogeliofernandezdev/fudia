package httpapi

import (
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	purchasedomain "github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
	"net/http"
)

type supplierView = purchasedomain.SupplierView
type supplierInput = purchasedomain.SupplierInput
type purchaseOrderItemInput = purchasedomain.PurchaseOrderItemInput
type purchaseOrderInput = purchasedomain.PurchaseOrderInput
type purchaseReceiptLineInput = purchasedomain.PurchaseReceiptLineInput
type purchaseReceiptInput = purchasedomain.PurchaseReceiptInput
type purchaseOrderSummary = purchasedomain.PurchaseOrderSummary
type purchaseOrderItemView = purchasedomain.PurchaseOrderItemView
type purchaseOrderDetail = purchasedomain.PurchaseOrderDetail
type purchaseReceiptSummary = purchasedomain.PurchaseReceiptSummary
type purchaseReceiptItemView = purchasedomain.PurchaseReceiptItemView
type purchaseReceiptDetail = purchasedomain.PurchaseReceiptDetail
type purchaseReturnInput = purchasedomain.PurchaseReturnInput

var normalizeSupplier = purchasedomain.NormalizeSupplier
var normalizePurchaseOrder = purchasedomain.NormalizePurchaseOrder

type purchaseInventoryItemInput = inventorydomain.PurchaseInventoryItemInput

func (a *API) listSuppliers(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.ListSuppliers(w, r)
}
func (a *API) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.ListPurchaseOrders(w, r)
}
func (a *API) listPurchaseReceipts(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.ListPurchaseReceipts(w, r)
}
func (a *API) getPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.GetPurchaseOrder(w, r)
}
func (a *API) getPurchaseReceipt(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.GetPurchaseReceipt(w, r)
}
func (a *API) createSupplier(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.CreateSupplier(w, r)
}
func (a *API) updateSupplier(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.UpdateSupplier(w, r)
}
func (a *API) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.CreatePurchaseOrder(w, r)
}
func (a *API) updatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.UpdatePurchaseOrder(w, r)
}
func (a *API) receivePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.ReceivePurchaseOrder(w, r)
}
func (a *API) createPurchaseReturn(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.CreatePurchaseReturn(w, r)
}
func (a *API) updateSupplierStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.UpdateSupplierStatus(w, r)
}
func (a *API) updatePurchaseOrderStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.UpdatePurchaseOrderStatus(w, r)
}
func (a *API) approvePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Purchasing.ApprovePurchaseOrder(w, r)
}
func (a *API) createPurchaseInventoryItem(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreatePurchaseInventoryItem(w, r)
}
