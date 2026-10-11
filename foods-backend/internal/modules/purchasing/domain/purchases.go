package domain

import (
	"math"
	"regexp"
	"strings"
	"time"
)

type SupplierView struct {
	ID     string `json:"id"`
	TaxID  string `json:"taxId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Active bool   `json:"active"`
}
type SupplierInput struct {
	TaxID string `json:"taxId"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}
type PurchaseOrderItemInput struct {
	InventoryItemID string  `json:"inventoryItemId"`
	PresentationID  string  `json:"presentationId"`
	Quantity        float64 `json:"quantity"`
	UnitCost        string  `json:"unitCost"`
}
type PurchaseOrderInput struct {
	SupplierID string                   `json:"supplierId"`
	ExpectedAt string                   `json:"expectedAt"`
	Notes      string                   `json:"notes"`
	Items      []PurchaseOrderItemInput `json:"items"`
}
type PurchaseReceiptLineInput struct {
	PurchaseOrderItemID string  `json:"purchaseOrderItemId"`
	Quantity            float64 `json:"quantity"`
}
type PurchaseReceiptInput struct {
	Notes          string                     `json:"notes"`
	IdempotencyKey string                     `json:"idempotencyKey"`
	Items          []PurchaseReceiptLineInput `json:"items"`
}
type PurchaseOrderSummary struct {
	ID           string     `json:"id"`
	Number       string     `json:"number"`
	SupplierID   string     `json:"supplierId"`
	SupplierName string     `json:"supplierName"`
	Status       string     `json:"status"`
	Total        string     `json:"total"`
	Notes        string     `json:"notes"`
	ExpectedAt   *string    `json:"expectedAt"`
	ItemCount    int        `json:"itemCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	ApprovedAt   *time.Time `json:"approvedAt"`
	ReceivedAt   *time.Time `json:"receivedAt"`
	CancelledAt  *time.Time `json:"cancelledAt"`
}
type PurchaseOrderItemView struct {
	ID                   string `json:"id"`
	InventoryItemID      string `json:"inventoryItemId"`
	ItemName             string `json:"itemName"`
	SKU                  string `json:"sku"`
	Unit                 string `json:"unit"`
	PresentationID       string `json:"presentationId"`
	PresentationType     string `json:"presentationType"`
	PresentationName     string `json:"presentationName"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
	Quantity             string `json:"quantity"`
	ReceivedQuantity     string `json:"receivedQuantity"`
	PendingQuantity      string `json:"pendingQuantity"`
	StockQuantity        string `json:"stockQuantity"`
	UnitCost             string `json:"unitCost"`
	LineTotal            string `json:"lineTotal"`
}
type PurchaseOrderDetail struct {
	PurchaseOrderSummary
	Items []PurchaseOrderItemView `json:"items"`
}
type PurchaseReceiptSummary struct {
	ID              string    `json:"id"`
	Code            string    `json:"code"`
	PurchaseOrderID string    `json:"purchaseOrderId"`
	Number          string    `json:"number"`
	SupplierName    string    `json:"supplierName"`
	Notes           string    `json:"notes"`
	CreatedByName   string    `json:"createdByName"`
	CreatedAt       time.Time `json:"createdAt"`
	ItemCount       int       `json:"itemCount"`
}
type PurchaseReceiptItemView struct {
	ID                   string `json:"id"`
	PurchaseOrderItemID  string `json:"purchaseOrderItemId"`
	InventoryItemID      string `json:"inventoryItemId"`
	ItemName             string `json:"itemName"`
	Quantity             string `json:"quantity"`
	PresentationType     string `json:"presentationType"`
	PresentationName     string `json:"presentationName"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
	StockQuantity        string `json:"stockQuantity"`
	UnitCost             string `json:"unitCost"`
	ReturnedQuantity     string `json:"returnedQuantity"`
	ReturnableQuantity   string `json:"returnableQuantity"`
}
type PurchaseReceiptDetail struct {
	PurchaseReceiptSummary
	Items []PurchaseReceiptItemView `json:"items"`
}
type PurchaseReturnInput struct {
	IdempotencyKey string            `json:"idempotencyKey"`
	Kind           string            `json:"kind"`
	Reason         string            `json:"reason"`
	Notes          string            `json:"notes"`
	Items          []ReturnLineInput `json:"items"`
}

var purchaseMoneyPattern = regexp.MustCompile(`^[0-9]+([.][0-9]{1,4})?$`)

func NormalizeSupplier(in SupplierInput) (SupplierInput, string) {
	in.Name = strings.TrimSpace(in.Name)
	in.TaxID = strings.TrimSpace(in.TaxID)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)
	if in.Name == "" {
		return in, "El nombre del proveedor es obligatorio."
	}
	if len(in.Name) > 180 || len(in.TaxID) > 11 || len(in.Email) > 180 || len(in.Phone) > 40 {
		return in, "Revisa la longitud de los datos del proveedor."
	}
	if in.Email != "" && !strings.Contains(in.Email, "@") {
		return in, "El correo del proveedor no es válido."
	}
	return in, ""
}
func NormalizePurchaseOrder(in PurchaseOrderInput) (PurchaseOrderInput, string) {
	in.SupplierID = strings.TrimSpace(in.SupplierID)
	in.ExpectedAt = strings.TrimSpace(in.ExpectedAt)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.SupplierID == "" {
		return in, "Selecciona un proveedor."
	}
	if in.ExpectedAt != "" {
		if _, err := time.Parse("2006-01-02", in.ExpectedAt); err != nil {
			return in, "La fecha esperada no es válida."
		}
	}
	if len(in.Notes) > 500 {
		return in, "Las notas no pueden superar 500 caracteres."
	}
	if len(in.Items) == 0 {
		return in, "Agrega al menos un artículo a la orden."
	}
	if len(in.Items) > 100 {
		return in, "Una orden no puede superar 100 líneas."
	}
	seen := map[string]bool{}
	for index := range in.Items {
		item := &in.Items[index]
		item.InventoryItemID = strings.TrimSpace(item.InventoryItemID)
		item.PresentationID = strings.TrimSpace(item.PresentationID)
		item.UnitCost = strings.TrimSpace(item.UnitCost)
		item.Quantity = math.Round(item.Quantity*1000) / 1000
		if item.InventoryItemID == "" || item.PresentationID == "" || item.Quantity <= 0 {
			return in, "Cada línea necesita artículo, presentación y cantidad mayor que cero."
		}
		if !purchaseMoneyPattern.MatchString(item.UnitCost) {
			return in, "Cada línea necesita un costo unitario válido."
		}
		key := item.InventoryItemID + ":" + item.PresentationID
		if seen[key] {
			return in, "No repitas el mismo artículo con la misma presentación."
		}
		seen[key] = true
	}
	return in, ""
}

type ReceivingLine struct {
	ID                   string
	InventoryItemID      string
	ProductID            *string
	ItemName             string
	PresentationID       string
	PresentationType     string
	UnitsPerPresentation float64
	OrderedQuantity      float64
	ReceivedQuantity     float64
	ReceiveQuantity      float64
	UnitCost             float64
}
type ReturnLineInput struct {
	PurchaseReceiptItemID string  `json:"purchaseReceiptItemId"`
	Quantity              float64 `json:"quantity"`
}
