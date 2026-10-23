package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"math"
	"sort"
	"strconv"
)

func (a *Stock) ApplyOrderQuantityDelta(ctx context.Context, s scope, orderID string, delta map[string]float64, movementType string) error {
	if len(delta) == 0 {
		return nil
	}
	var businessDate string
	if err := func() error { var err error; businessDate, err = a.repo.OrderBusinessDate(ctx, s, orderID); return err }(); err != nil {
		return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos determinar la fecha operativa del pedido.")
	}

	productIDs := make([]string, 0, len(delta))
	for id := range delta {
		productIDs = append(productIDs, id)
	}
	sort.Strings(productIDs)

	for _, productID := range productIDs {
		change := delta[productID]
		var name, control string
		if err := func() error {
			var err error
			name, control, err = a.repo.LockQuantityProduct(ctx, s, productID)
			return err
		}(); err != nil {
			if errors.Is(err, ErrNotFound) {
				return stockFailure(apperror.Conflict, "product_unavailable", "Un producto del pedido ya no existe.")
			}
			return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos validar la cantidad del producto.")
		}

		switch control {
		case "none":
			continue
		case "portions":
			if math.Abs(change-math.Round(change)) > 0.000001 {
				return stockFailure(apperror.Invalid, "invalid_portion_quantity", name+" solo se vende por porciones enteras.")
			}
			var portionQuantity *int
			var soldQuantity int
			var manualStatus string
			err := func() error {
				var err error
				portionQuantity, soldQuantity, manualStatus, err = a.repo.LockPortions(ctx, s, productID, businessDate)
				return err
			}()
			if errors.Is(err, ErrNotFound) {
				return stockFailure(apperror.Conflict, "portions_unavailable", name+" no tiene porciones cargadas para este día.")
			}
			if err != nil {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos validar las porciones disponibles.")
			}
			step := int(math.Round(change))
			nextSold := soldQuantity + step
			if step > 0 && manualStatus == "sold_out" {
				return stockFailure(apperror.Conflict, "insufficient_portions", name+" está agotado por hoy.")
			}
			if step > 0 && (portionQuantity == nil || nextSold > *portionQuantity) {
				left := 0
				if portionQuantity != nil && *portionQuantity > soldQuantity {
					left = *portionQuantity - soldQuantity
				}
				return stockFailure(apperror.Conflict, "insufficient_portions", "La cantidad excede las existencias: solo quedan "+strconv.Itoa(left)+" porciones de "+name+".")
			}
			if nextSold < 0 {
				return stockFailure(apperror.Unavailable, "quantity_inconsistent", "No pudimos reconciliar las porciones del pedido.")
			}
			if _, err := a.repo.UpdateSoldPortions(ctx, s, productID, businessDate, nextSold); err != nil {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos actualizar las porciones disponibles.")
			}
		case "inventory":
			var inventoryItemID string
			err := func() error {
				var err error
				inventoryItemID, err = a.repo.LockProductInventory(ctx, s, productID)
				return err
			}()
			if errors.Is(err, ErrNotFound) {
				return stockFailure(apperror.Conflict, "inventory_not_linked", name+" todavía no tiene inventario registrado.")
			}
			if err != nil {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos validar el inventario del producto.")
			}
			if _, err := a.repo.EnsureEntryBalance(ctx, s, inventoryItemID); err != nil {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos preparar el saldo del producto.")
			}
			var balance, averageCost float64
			if err := func() error {
				var err error
				balance, averageCost, err = a.repo.LockEntryBalance(ctx, s, inventoryItemID)
				return err
			}(); err != nil {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos bloquear el saldo del producto.")
			}
			balanceDelta := -change
			nextBalance := balance + balanceDelta
			if nextBalance < -0.000001 {
				left := strconv.FormatFloat(math.Max(0, balance), 'f', -1, 64)
				return stockFailure(apperror.Conflict, "insufficient_stock", "La cantidad excede las existencias: solo quedan "+left+" unidades de "+name+".")
			}
			if nextBalance < 0 {
				nextBalance = 0
			}
			nextAverage := averageCost
			if _, err := a.repo.SaveBalance(ctx, s, inventoryItemID, nextBalance, nextAverage); err != nil {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos actualizar el stock del producto.")
			}
			if math.Abs(balanceDelta) > 0.000001 {
				if err := a.movement(ctx, s, &productID, inventoryItemID, movementType, balanceDelta, nextBalance, averageCost, "order", orderID, orderMovementNote(movementType)); err != nil {
					return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos registrar el movimiento de Kárdex.")
				}
			}
		default:
			return stockFailure(apperror.Unavailable, "quantity_control_invalid", "El producto tiene un control de cantidad inválido.")
		}
	}
	return nil
}

// orderMovementNote describes an order movement; the Kárdex shows the order code
// from source_id, so the note never repeats the raw identifier.
func orderMovementNote(movementType string) string {
	switch movementType {
	case "sale_reversal":
		return "Reversa de venta"
	case "sale_adjustment":
		return "Ajuste por edición del pedido"
	default:
		return "Salida automática por venta"
	}
}
func (a *Stock) DesiredRecipeUsage(ctx context.Context, s scope, products map[string]float64) (map[string]float64, error) {
	recipes, inventory, err := a.repo.RecipeModules(ctx, s)
	if err != nil {
		return nil, err
	}
	out := map[string]float64{}
	if !recipes || !inventory {
		return out, nil
	}
	ids := make([]string, 0, len(products))
	for id := range products {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		recipeID, yield, err := a.repo.ActiveRecipe(ctx, s, id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		ingredients, err := a.repo.RecipeIngredients(ctx, s, recipeID)
		if err != nil {
			return nil, err
		}
		for _, x := range ingredients {
			need := products[id] / yield * x.Quantity * (1 + x.WastePercent/100)
			out[x.ItemID] += math.Round(need*1000) / 1000
		}
	}
	return out, nil
}
func (a *Stock) NetRecipeUsage(ctx context.Context, s scope, orderID string) (map[string]float64, error) {
	return a.repo.NetRecipeUsage(ctx, s, orderID)
}
func (a *Stock) ApplyRecipeUsageDelta(ctx context.Context, s scope, orderID string, delta map[string]float64) error {
	ids := make([]string, 0, len(delta))
	for id := range delta {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, itemID := range ids {
		change := math.Round(delta[itemID]*1000) / 1000
		if math.Abs(change) <= 0.000001 {
			continue
		}
		bal, err := a.repo.LockBalance(ctx, s, itemID)
		if errors.Is(err, ErrNotFound) {
			return stockFailure(apperror.Conflict, "recipe_item_unavailable", "Un insumo de receta ya no está disponible.")
		}
		if err != nil {
			return stockFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el stock de receta.")
		}
		if change > bal.Quantity+0.000001 {
			return stockFailure(apperror.Conflict, "insufficient_recipe_stock", "No hay stock suficiente de "+bal.Name+" para preparar el pedido.")
		}
		after := math.Round((bal.Quantity-change)*1000) / 1000
		avg := bal.AverageUnitCost
		if _, err = a.repo.SaveBalance(ctx, s, itemID, after, avg); err != nil {
			return stockFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos actualizar el stock de receta.")
		}
		movement := "recipe_consumption"
		note := "Consumo automático por receta"
		if change < 0 {
			movement = "recipe_reversal"
			note = "Reversa de consumo por receta"
		}
		if err = a.movement(ctx, s, bal.ProductID, itemID, movement, -change, after, bal.AverageUnitCost, "order", orderID, note); err != nil {
			return stockFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos registrar el Kárdex de receta.")
		}
	}
	return nil
}
