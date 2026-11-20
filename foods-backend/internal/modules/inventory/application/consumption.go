package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"math"
	"sort"
	"strconv"
)

// StockWrites agrupa lo que el consumo de un pedido guarda al final, en una sola
// ida y vuelta y en este orden: porciones vendidas, saldos y movimientos de Kárdex.
type StockWrites struct {
	Portions  []PortionWrite
	Balances  []BalanceWrite
	Movements []Movement
}

type PortionWrite struct {
	ProductID    string
	BusinessDate string
	SoldQuantity int
}

type BalanceWrite struct {
	ItemID      string
	Quantity    float64
	AverageCost float64
}

// ApplyOrderQuantityDelta descuenta o devuelve porciones y stock de los productos
// del pedido. Bloquea primero los productos, luego todas sus porciones del día,
// sus artículos de inventario y sus saldos, cada grupo en una consulta ordenada
// por id; valida producto por producto en orden de id, con los mismos mensajes que
// el ciclo anterior, y guarda todo junto al final.
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
	locked, err := a.repo.LockQuantityProducts(ctx, s, productIDs)
	if err != nil {
		return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos validar la cantidad del producto.")
	}

	var portionIDs, inventoryIDs []string
	for _, productID := range productIDs {
		product, ok := locked[productID]
		if !ok {
			continue
		}
		switch product.Control {
		case "portions":
			portionIDs = append(portionIDs, productID)
		case "inventory":
			inventoryIDs = append(inventoryIDs, productID)
		}
	}
	portions, err := a.repo.LockPortionsFor(ctx, s, portionIDs, businessDate)
	if err != nil {
		return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos validar las porciones disponibles.")
	}
	items, err := a.repo.LockProductInventories(ctx, s, inventoryIDs)
	if err != nil {
		return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos validar el inventario del producto.")
	}
	itemIDs := make([]string, 0, len(items))
	for _, itemID := range items {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	balances, err := a.repo.LockBalances(ctx, s, itemIDs)
	if err != nil {
		return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos bloquear el saldo del producto.")
	}

	writes := StockWrites{}
	for _, productID := range productIDs {
		change := delta[productID]
		product, ok := locked[productID]
		if !ok {
			return stockFailure(apperror.Conflict, "product_unavailable", "Un producto del pedido ya no existe.")
		}
		name, control := product.Name, product.Control

		switch control {
		case "none":
			continue
		case "portions":
			if math.Abs(change-math.Round(change)) > 0.000001 {
				return stockFailure(apperror.Invalid, "invalid_portion_quantity", name+" solo se vende por porciones enteras.")
			}
			portion, found := portions[productID]
			if !found {
				return stockFailure(apperror.Conflict, "portions_unavailable", name+" no tiene porciones cargadas para este día.")
			}
			step := int(math.Round(change))
			nextSold := portion.SoldQuantity + step
			if step > 0 && portion.ManualStatus == "sold_out" {
				return stockFailure(apperror.Conflict, "insufficient_portions", name+" está agotado por hoy.")
			}
			if step > 0 && (portion.PortionQuantity == nil || nextSold > *portion.PortionQuantity) {
				left := 0
				if portion.PortionQuantity != nil && *portion.PortionQuantity > portion.SoldQuantity {
					left = *portion.PortionQuantity - portion.SoldQuantity
				}
				return stockFailure(apperror.Conflict, "insufficient_portions", "La cantidad excede las existencias: solo quedan "+strconv.Itoa(left)+" porciones de "+name+".")
			}
			if nextSold < 0 {
				return stockFailure(apperror.Unavailable, "quantity_inconsistent", "No pudimos reconciliar las porciones del pedido.")
			}
			writes.Portions = append(writes.Portions, PortionWrite{ProductID: productID, BusinessDate: businessDate, SoldQuantity: nextSold})
		case "inventory":
			inventoryItemID, found := items[productID]
			if !found {
				return stockFailure(apperror.Conflict, "inventory_not_linked", name+" todavía no tiene inventario registrado.")
			}
			current, found := balances[inventoryItemID]
			if !found {
				return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos bloquear el saldo del producto.")
			}
			balance, averageCost := current.Quantity, current.AverageUnitCost
			balanceDelta := -change
			nextBalance := balance + balanceDelta
			if nextBalance < -0.000001 {
				left := strconv.FormatFloat(math.Max(0, balance), 'f', -1, 64)
				return stockFailure(apperror.Conflict, "insufficient_stock", "La cantidad excede las existencias: solo quedan "+left+" unidades de "+name+".")
			}
			if nextBalance < 0 {
				nextBalance = 0
			}
			writes.Balances = append(writes.Balances, BalanceWrite{ItemID: inventoryItemID, Quantity: nextBalance, AverageCost: averageCost})
			if math.Abs(balanceDelta) > 0.000001 {
				productRef := productID
				writes.Movements = append(writes.Movements, valuedMovement(&productRef, inventoryItemID, movementType, balanceDelta, nextBalance, averageCost, averageCost, "order", orderID, orderMovementNote(movementType)))
			}
		default:
			return stockFailure(apperror.Unavailable, "quantity_control_invalid", "El producto tiene un control de cantidad inválido.")
		}
	}
	if err := a.repo.ApplyStockWrites(ctx, s, writes); err != nil {
		return stockFailure(apperror.Unavailable, "quantity_unavailable", "No pudimos actualizar el stock del producto.")
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
	active, err := a.repo.ActiveRecipes(ctx, s, ids)
	if err != nil {
		return nil, err
	}
	recipeIDs := make([]string, 0, len(active))
	for _, id := range ids {
		if recipe, ok := active[id]; ok {
			recipeIDs = append(recipeIDs, recipe.ID)
		}
	}
	// Los insumos de todas las recetas en una consulta (antes, una por producto).
	ingredients, err := a.repo.RecipeIngredientsFor(ctx, s, recipeIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		recipe, ok := active[id]
		if !ok {
			continue
		}
		yield := recipe.Yield
		for _, x := range ingredients[recipe.ID] {
			need := products[id] / yield * x.Quantity * (1 + x.WastePercent/100)
			out[x.ItemID] += need
		}
	}
	for id, quantity := range out {
		out[id] = domain.RoundStockQuantity(quantity)
	}
	return out, nil
}
func (a *Stock) NetRecipeUsage(ctx context.Context, s scope, orderID string) (map[string]float64, error) {
	return a.repo.NetRecipeUsage(ctx, s, orderID)
}

// ApplyRecipeUsageDelta consume o devuelve los insumos de receta del pedido:
// bloquea todos sus saldos en una consulta ordenada por id, valida insumo por
// insumo en ese orden y guarda saldos y Kárdex en una sola ida y vuelta.
func (a *Stock) ApplyRecipeUsageDelta(ctx context.Context, s scope, orderID string, delta map[string]float64) error {
	ids := make([]string, 0, len(delta))
	for id := range delta {
		if domain.RoundStockQuantity(delta[id]) != 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Strings(ids)
	balances, err := a.repo.LockBalances(ctx, s, ids)
	if err != nil {
		return stockFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos validar el stock de receta.")
	}
	writes := StockWrites{}
	for _, itemID := range ids {
		change := domain.RoundStockQuantity(delta[itemID])
		bal, ok := balances[itemID]
		if !ok {
			return stockFailure(apperror.Conflict, "recipe_item_unavailable", "Un insumo de receta ya no está disponible.")
		}
		if change > bal.Quantity+1e-9 {
			return stockFailure(apperror.Conflict, "insufficient_recipe_stock", "No hay stock suficiente de "+bal.Name+" para preparar el pedido.")
		}
		after := domain.RoundStockQuantity(bal.Quantity - change)
		writes.Balances = append(writes.Balances, BalanceWrite{ItemID: itemID, Quantity: after, AverageCost: bal.AverageUnitCost})
		movement := "recipe_consumption"
		note := "Consumo automático por receta"
		if change < 0 {
			movement = "recipe_reversal"
			note = "Reversa de consumo por receta"
		}
		writes.Movements = append(writes.Movements, valuedMovement(bal.ProductID, itemID, movement, -change, after, bal.AverageUnitCost, bal.AverageUnitCost, "order", orderID, note))
	}
	if err := a.repo.ApplyStockWrites(ctx, s, writes); err != nil {
		return stockFailure(apperror.Unavailable, "recipe_inventory_unavailable", "No pudimos registrar el Kárdex de receta.")
	}
	return nil
}
