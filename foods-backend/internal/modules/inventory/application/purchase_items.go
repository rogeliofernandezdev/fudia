package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

func (a *Stock) CreatePurchaseItem(ctx context.Context, s scope, input domain.PurchaseInventoryItemInput) (domain.InventoryProductOption, error) {
	var result domain.InventoryProductOption
	legacyType, legacyFactor := input.PresentationType, input.UnitsPerPresentation
	if input.Presentations != nil {
		if len(input.Presentations) == 0 {
			return result, stockFailure(apperror.Invalid, "invalid_inventory_presentation", "Agrega al menos una presentación.")
		}
		legacyType, legacyFactor = "unit", 1
	}
	normalized, invalid := domain.NormalizeInventoryEntry(domain.InventoryEntryInput{
		ProductID:            input.ProductID,
		NewProduct:           input.NewProduct,
		NewIngredient:        input.NewIngredient,
		Quantity:             1,
		Unit:                 input.Unit,
		PresentationType:     legacyType,
		UnitsPerPresentation: legacyFactor,
		MinimumStock:         &input.MinimumStock,
	})
	if invalid != "" {
		return result, stockFailure(apperror.Invalid, "invalid_purchase_item", invalid)
	}
	if normalized.ProductID == "" && normalized.NewProduct == nil && normalized.NewIngredient == nil {
		return result, stockFailure(apperror.Invalid, "invalid_purchase_item", "Selecciona un producto de la carta o crea un artículo nuevo.")
	}
	presentations := input.Presentations
	if len(presentations) == 0 {
		presentations = []domain.InventoryPresentationInput{{PresentationType: normalized.PresentationType, UnitsPerPresentation: normalized.UnitsPerPresentation, IsDefault: true}}
	}
	if len(presentations) > 50 {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_presentation", "Un artículo admite hasta 50 presentaciones.")
	}
	defaults := 0
	seen := map[string]bool{}
	for i, p := range presentations {
		p, invalid = domain.NormalizeInventoryPresentation(p)
		key := fmt.Sprintf("%s:%0.3f", p.PresentationType, p.UnitsPerPresentation)
		if invalid != "" || seen[key] {
			return result, stockFailure(apperror.Invalid, "invalid_inventory_presentation", "Revisa las conversiones; no repitas presentaciones.")
		}
		seen[key] = true
		presentations[i] = p
		if p.IsDefault {
			defaults++
		}
	}
	if defaults > 1 {
		return result, stockFailure(apperror.Invalid, "invalid_inventory_presentation", "Selecciona una sola presentación predeterminada.")
	}
	if defaults == 0 {
		presentations[0].IsDefault = true
	}

	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos crear el artículo.")
	}
	defer tx.Rollback(ctx)

	var created domain.InventoryCatalogCreateResult
	var createErr error
	if normalized.ProductID != "" {
		created, createErr = a.linkExistingProduct(tx.Context(), s, normalized)
	} else {
		created, createErr = a.CreateCatalogItem(tx.Context(), s, normalized)
	}
	if createErr != nil {
		return result, createErr
	}
	for _, p := range presentations {
		var id string
		id, err = a.ensurePresentation(tx.Context(), s, created.InventoryItemID, p.PresentationType, p.UnitsPerPresentation)
		var comboErr *inventoryCombinationError
		if errors.As(err, &comboErr) {
			return result, stockFailure(apperror.Invalid, comboErr.Code(), comboErr.Error())
		}
		if err == nil && p.IsDefault {
			err = a.repo.DefaultPresentation(tx.Context(), s, created.InventoryItemID, id)
		}
		if err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos guardar las presentaciones del artículo.")
		}
	}
	if _, err = a.repo.EnsureEntryBalance(tx.Context(), s, created.InventoryItemID); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos preparar el artículo con stock cero.")
	}

	presentationOptions, err := a.repo.Presentations(tx.Context(), s, created.InventoryItemID)
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos cargar las presentaciones del artículo.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos crear el artículo.")
	}
	if normalized.ProductID != "" {
		_ = a.audit.Record(ctx, s, "product.linked_to_inventory", "product", *created.ProductID)
	} else if created.ProductID != nil {
		_ = a.audit.Record(ctx, s, "product.created_from_purchase", "product", *created.ProductID)
	} else {
		_ = a.audit.Record(ctx, s, "inventory.ingredient_created_from_purchase", "inventory_item", created.InventoryItemID)
	}
	var quantityControl *string
	if created.ProductID != nil {
		control := "inventory"
		quantityControl = &control
	}
	minimumStock := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", normalized.MinimumStockValue()), "0"), ".")
	if minimumStock == "" {
		minimumStock = "0"
	}
	return domain.InventoryProductOption{
		ID:              created.InventoryItemID,
		ProductID:       created.ProductID,
		SKU:             created.SKU,
		Name:            created.Name,
		Kind:            created.Kind,
		CategoryName:    nil,
		QuantityControl: quantityControl,
		Unit:            created.Unit,
		Quantity:        "0",
		MinimumStock:    minimumStock,
		Presentations:   presentationOptions,
	}, nil

}
