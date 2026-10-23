package application

import (
	"context"
	"errors"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// LinkableProducts lists catalog products that still use Disponibilidad (or no
// control) so Compras can link them instead of creating a duplicate product.
func (a *Stock) LinkableProducts(ctx context.Context, s scope, query string) ([]domain.LinkableProduct, error) {
	out, err := a.repo.LinkableProducts(ctx, s, strings.TrimSpace(query), 20)
	if err != nil {
		return nil, stockFailure(apperror.Unavailable, "linkable_products_unavailable", "No pudimos cargar los productos de la carta.")
	}
	return out, nil
}

// linkExistingProduct switches an existing product to physical inventory and
// creates (or reuses) its inventory item. It never creates stock: the balance
// starts at zero and grows only through receipts or inventory documents.
// It runs inside the caller's transaction.
func (a *Stock) linkExistingProduct(ctx context.Context, s scope, in domain.InventoryEntryInput) (domain.InventoryCatalogCreateResult, error) {
	result := domain.InventoryCatalogCreateResult{Unit: in.Unit, Kind: "product"}
	if invalid := a.validateUnit(ctx, s, in.Unit); invalid != nil {
		return result, invalid
	}
	product, err := a.repo.LockLinkableProduct(ctx, s, in.ProductID)
	if errors.Is(err, ErrNotFound) {
		return result, stockFailure(apperror.NotFound, "product_not_found", "El producto no existe o está inactivo.")
	}
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar el producto.")
	}
	if product.IsCombo {
		return result, stockFailure(apperror.Conflict, "product_link_combo", "Los combos descuentan el stock de sus componentes; no se vinculan al inventario.")
	}
	if product.HasRecipe {
		return result, stockFailure(apperror.Conflict, "product_link_recipe", "Este producto tiene receta: su stock ya se descuenta por sus ingredientes.")
	}
	if product.QuantityControl != "inventory" {
		open, err := a.repo.ProductInOpenOrders(ctx, s, product.ID)
		if err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos validar los pedidos abiertos del producto.")
		}
		if open {
			return result, stockFailure(apperror.Conflict, "quantity_control_open_orders", "Cierra o cancela los pedidos abiertos de este producto antes de controlarlo con inventario.")
		}
		if err = a.repo.MarkProductInventoryControlled(ctx, s, product); err != nil {
			return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos cambiar el control de cantidad del producto.")
		}
	}
	itemID, unit, err := a.repo.LinkEntryProduct(ctx, s, in, product.ID, product.Name)
	if err != nil {
		return result, stockFailure(apperror.Unavailable, "inventory_unavailable", "No pudimos vincular el producto al inventario.")
	}
	if unit != in.Unit {
		return result, stockFailure(apperror.Conflict, "inventory_unit_conflict", "La unidad de inventario de este producto es "+unit+".")
	}
	productID := product.ID
	result.InventoryItemID, result.ProductID, result.SKU, result.Name, result.Unit = itemID, &productID, product.SKU, product.Name, unit
	return result, nil
}
