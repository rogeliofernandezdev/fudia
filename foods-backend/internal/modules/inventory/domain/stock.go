package domain

import (
	"math"
	"strings"
	"time"
)

type InventoryItemView struct {
	InventoryItemID string    `json:"inventoryItemId"`
	ProductID       *string   `json:"productId"`
	SKU             string    `json:"sku"`
	Name            string    `json:"name"`
	Kind            string    `json:"kind"`
	CategoryName    *string   `json:"categoryName"`
	Active          bool      `json:"active"`
	Unit            string    `json:"unit"`
	Quantity        string    `json:"quantity"`
	MinimumStock    string    `json:"minimumStock"`
	ReorderPoint    string    `json:"reorderPoint"`
	OptimalStock    string    `json:"optimalStock"`
	AverageUnitCost string    `json:"averageUnitCost"`
	StockValue      string    `json:"stockValue"`
	Status          string    `json:"status"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
type InventoryPresentationOption struct {
	ID                   string `json:"id"`
	PresentationType     string `json:"presentationType"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
	Name                 string `json:"name"`
	IsDefault            bool   `json:"isDefault"`
}
type InventoryProductOption struct {
	ID              string                        `json:"id"`
	ProductID       *string                       `json:"productId"`
	SKU             string                        `json:"sku"`
	Name            string                        `json:"name"`
	Kind            string                        `json:"kind"`
	CategoryName    *string                       `json:"categoryName"`
	QuantityControl *string                       `json:"quantityControl"`
	Unit            string                        `json:"unit"`
	Quantity        string                        `json:"quantity"`
	MinimumStock    string                        `json:"minimumStock"`
	ReorderPoint    string                        `json:"reorderPoint"`
	OptimalStock    string                        `json:"optimalStock"`
	AverageUnitCost string                        `json:"averageUnitCost"`
	Presentations   []InventoryPresentationOption `json:"presentations"`
}
type InventoryNewProductInput struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CategoryID  *string `json:"categoryId"`
	Price       string  `json:"price"`
}
type InventoryNewIngredientInput struct {
	Name string `json:"name"`
}
type InventoryEntryInput struct {
	InventoryItemID      string                       `json:"inventoryItemId"`
	ProductID            string                       `json:"productId"`
	NewProduct           *InventoryNewProductInput    `json:"newProduct"`
	NewIngredient        *InventoryNewIngredientInput `json:"newIngredient"`
	Quantity             float64                      `json:"quantity"`
	Unit                 string                       `json:"unit"`
	PresentationType     string                       `json:"presentationType"`
	UnitsPerPresentation float64                      `json:"unitsPerPresentation"`
	MinimumStock         float64                      `json:"minimumStock"`
	Note                 string                       `json:"note"`
}
type InventoryCatalogCreateResult struct {
	InventoryItemID string
	ProductID       *string
	SKU             string
	Name            string
	Unit            string
	Kind            string
}
type InventoryMovementView struct {
	ID                string    `json:"id"`
	InventoryItemID   string    `json:"inventoryItemId"`
	ProductID         *string   `json:"productId"`
	ItemName          string    `json:"itemName"`
	MovementType      string    `json:"movementType"`
	AdjustmentType    *string   `json:"adjustmentType,omitempty"`
	Reason            *string   `json:"reason,omitempty"`
	QuantityDelta     string    `json:"quantityDelta"`
	BalanceBefore     string    `json:"balanceBefore"`
	BalanceAfter      string    `json:"balanceAfter"`
	UnitCost          string    `json:"unitCost"`
	ValueDelta        string    `json:"valueDelta"`
	BalanceValueAfter string    `json:"balanceValueAfter"`
	SourceType        string    `json:"sourceType"`
	SourceID          string    `json:"sourceId"`
	SourceReference   string    `json:"sourceReference"`
	Note              string    `json:"note"`
	CreatedByName     string    `json:"createdByName"`
	CreatedAt         time.Time `json:"createdAt"`
}
type InventoryAdjustmentInput struct {
	InventoryItemID string  `json:"inventoryItemId"`
	MovementType    string  `json:"movementType"`
	Reason          string  `json:"reason"`
	Quantity        float64 `json:"quantity"`
	Observation     string  `json:"observation"`
}
type InventoryCombinationView struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	UnitName  string `json:"unitName"`
	IsDefault bool   `json:"isDefault"`
}
type InventoryCombinationInput struct {
	Unit      string `json:"unit"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}
type InventoryPresentationInput struct {
	PresentationType     string  `json:"presentationType"`
	UnitsPerPresentation float64 `json:"unitsPerPresentation"`
	IsDefault            bool    `json:"isDefault"`
}
type InventoryBalanceLock struct {
	ProductID       *string
	Name            string
	Unit            string
	Quantity        float64
	AverageUnitCost float64
}

func NormalizeInventoryEntry(in InventoryEntryInput) (InventoryEntryInput, string) {
	in.InventoryItemID = strings.TrimSpace(in.InventoryItemID)
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.Unit = strings.TrimSpace(in.Unit)
	in.PresentationType = strings.ToLower(strings.TrimSpace(in.PresentationType))
	in.Note = strings.TrimSpace(in.Note)
	if math.IsNaN(in.Quantity) || math.IsInf(in.Quantity, 0) || in.Quantity <= 0 || in.Quantity >= 1e11 {
		return in, "La cantidad de entrada debe ser mayor que cero."
	}
	in.Quantity = math.Round(in.Quantity*1000) / 1000
	in.UnitsPerPresentation = math.Round(in.UnitsPerPresentation*1000) / 1000
	if in.Quantity <= 0 {
		return in, "La cantidad de entrada debe ser al menos 0.001."
	}
	if math.IsNaN(in.MinimumStock) || math.IsInf(in.MinimumStock, 0) || in.MinimumStock < 0 || in.MinimumStock >= 1e11 {
		return in, "El stock mínimo no puede ser negativo."
	}
	if in.Unit == "" {
		in.Unit = "und"
	}
	if in.PresentationType == "" {
		in.PresentationType = "unit"
	}
	if in.PresentationType == "unit" {
		in.UnitsPerPresentation = 1
	} else {
		if math.Abs(in.Quantity-math.Round(in.Quantity)) > 0.000001 {
			return in, "La cantidad de presentaciones debe ser un número entero. Para fracciones, registra unidades base."
		}
	}
	if _, invalid := NormalizeInventoryPresentation(InventoryPresentationInput{PresentationType: in.PresentationType, UnitsPerPresentation: in.UnitsPerPresentation}); invalid != "" {
		return in, invalid
	}

	sourceCount := 0
	if in.InventoryItemID != "" {
		sourceCount++
	}
	if in.ProductID != "" {
		sourceCount++
	}
	if in.NewProduct != nil {
		sourceCount++
	}
	if in.NewIngredient != nil {
		sourceCount++
	}
	if sourceCount != 1 {
		return in, "Selecciona un artículo existente, crea un producto vendible o crea un insumo."
	}

	if in.NewProduct != nil {
		in.NewProduct.Name = strings.TrimSpace(in.NewProduct.Name)
		in.NewProduct.SKU = strings.TrimSpace(in.NewProduct.SKU)
		in.NewProduct.Description = strings.TrimSpace(in.NewProduct.Description)
		in.NewProduct.Price = strings.TrimSpace(in.NewProduct.Price)
		if in.NewProduct.Name == "" || in.NewProduct.Price == "" {
			return in, "El nuevo producto vendible necesita nombre y precio de venta."
		}
		if in.NewProduct.CategoryID == nil || strings.TrimSpace(*in.NewProduct.CategoryID) == "" {
			return in, "Selecciona una categoría para la mercadería vendible."
		}
		categoryID := strings.TrimSpace(*in.NewProduct.CategoryID)
		in.NewProduct.CategoryID = &categoryID
	}
	if in.NewIngredient != nil {
		in.NewIngredient.Name = strings.TrimSpace(in.NewIngredient.Name)
		if in.NewIngredient.Name == "" {
			return in, "El nuevo insumo necesita un nombre."
		}
	}
	return in, ""
}
func ValidInventoryAdjustmentReason(movementType, reason string) bool {
	switch movementType {
	case "entry":
		return reason == "surplus_adjustment"
	case "exit":
		return reason == "shortage_adjustment" || reason == "waste" || reason == "expiration" || reason == "other_exit"
	default:
		return false
	}
}
func NormalizeInventoryAdjustment(in InventoryAdjustmentInput) (InventoryAdjustmentInput, string) {
	in.InventoryItemID = strings.TrimSpace(in.InventoryItemID)
	in.MovementType = strings.ToLower(strings.TrimSpace(in.MovementType))
	in.Reason = strings.ToLower(strings.TrimSpace(in.Reason))
	in.Observation = strings.TrimSpace(in.Observation)
	in.Quantity = math.Round(in.Quantity*1000) / 1000
	if in.InventoryItemID == "" {
		return in, "Selecciona un artículo existente."
	}
	if in.MovementType != "entry" && in.MovementType != "exit" {
		return in, "Selecciona si el ajuste es una entrada o una salida."
	}
	if !ValidInventoryAdjustmentReason(in.MovementType, in.Reason) {
		return in, "Selecciona un motivo válido para el tipo de movimiento."
	}
	if in.Quantity <= 0 {
		return in, "La cantidad debe ser mayor que cero."
	}
	if len(in.Observation) > 240 {
		return in, "La observación no puede superar 240 caracteres."
	}
	return in, ""
}
func NormalizeInventoryPresentation(in InventoryPresentationInput) (InventoryPresentationInput, string) {
	in.PresentationType = strings.ToLower(strings.TrimSpace(in.PresentationType))
	if !ValidUnitCode(in.PresentationType) {
		return in, "Selecciona una presentación válida."
	}
	if math.IsNaN(in.UnitsPerPresentation) || math.IsInf(in.UnitsPerPresentation, 0) || in.UnitsPerPresentation <= 0 || in.UnitsPerPresentation >= 1e11 {
		return in, "La conversión debe ser mayor que cero y menor que 100000000000."
	}
	if math.Abs(in.UnitsPerPresentation*1000-math.Round(in.UnitsPerPresentation*1000)) > 0.000001 {
		return in, "Usa como máximo tres decimales en la conversión."
	}
	if in.PresentationType == "unit" && in.UnitsPerPresentation != 1 {
		return in, "La unidad base equivale a una unidad."
	}
	return in, ""
}
func ValidDateFilters(values ...string) bool {
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return false
		}
	}
	return true
}

type LocationSettings struct {
	MinimumStock float64 `json:"minimumStock"`
	ReorderPoint float64 `json:"reorderPoint"`
	OptimalStock float64 `json:"optimalStock"`
}

type PurchaseInventoryItemInput struct {
	NewProduct           *InventoryNewProductInput    `json:"newProduct"`
	NewIngredient        *InventoryNewIngredientInput `json:"newIngredient"`
	Unit                 string                       `json:"unit"`
	PresentationType     string                       `json:"presentationType"`
	UnitsPerPresentation float64                      `json:"unitsPerPresentation"`
	MinimumStock         float64                      `json:"minimumStock"`
	Presentations        []InventoryPresentationInput `json:"presentations"`
}
