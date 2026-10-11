package httpapi

import (
	"context"
	"errors"
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/jackc/pgx/v5"
	"math"
	"net/http"
	"sort"
)

type recipeItemView = menudomain.RecipeItemView
type recipeView = menudomain.RecipeView
type recipeInput = menudomain.RecipeInput
type moduleRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func moduleEnabled(ctx context.Context, q moduleRowQuerier, organizationID, key string) (bool, error) {
	var active bool
	err := q.QueryRow(ctx, `SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key=$2`, organizationID, key).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	return active, err
}
func recipesRuntimeEnabled(ctx context.Context, q moduleRowQuerier, organizationID string) (bool, error) {
	recipes, err := moduleEnabled(ctx, q, organizationID, "recetas")
	if err != nil || !recipes {
		return recipes, err
	}
	inventory, err := moduleEnabled(ctx, q, organizationID, "inventario")
	if err != nil {
		return false, err
	}
	return inventory, nil
}
func preparedRecipeProductUsage(items []preparedOrderItem) map[string]float64 {
	usage := map[string]float64{}
	for _, item := range items {
		if item.ProductID != nil {
			usage[*item.ProductID] += item.Qty
		}
		if item.ItemType == "combo" {
			for _, sel := range item.Selections {
				usage[sel.ProductID] += item.Qty
			}
		}
	}
	return usage
}
func desiredRecipeInventoryUsage(ctx context.Context, tx pgx.Tx, s scope, items []preparedOrderItem) (map[string]float64, error) {
	enabled, err := recipesRuntimeEnabled(ctx, tx, s.OrganizationID)
	if err != nil {
		return nil, err
	}
	out := map[string]float64{}
	if !enabled {
		return out, nil
	}
	products := preparedRecipeProductUsage(items)
	ids := make([]string, 0, len(products))
	for id := range products {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, productID := range ids {
		sold := products[productID]
		var recipeID string
		var yield float64
		err = tx.QueryRow(ctx, `SELECT id,yield_quantity::float8 FROM product_recipes WHERE organization_id=$1 AND product_id=$2 AND active`, s.OrganizationID, productID).Scan(&recipeID, &yield)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		rows, err := tx.Query(ctx, `SELECT inventory_item_id::text,quantity::float8,waste_percent::float8 FROM product_recipe_items WHERE recipe_id=$1 AND organization_id=$2 ORDER BY inventory_item_id`, recipeID, s.OrganizationID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var itemID string
			var qty, waste float64
			if err = rows.Scan(&itemID, &qty, &waste); err != nil {
				rows.Close()
				return nil, err
			}
			need := sold / yield * qty * (1 + waste/100)
			out[itemID] += math.Round(need*1000) / 1000
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func loadNetRecipeUsage(ctx context.Context, tx pgx.Tx, s scope, orderID string) (map[string]float64, error) {
	rows, err := tx.Query(ctx, `
		SELECT inventory_item_id::text,COALESCE(sum(-quantity_delta),0)::float8
		FROM stock_movements
		WHERE organization_id=$1 AND location_id=$2 AND source_type='order' AND source_id=$3
		  AND movement_type IN ('recipe_consumption','recipe_reversal')
		GROUP BY inventory_item_id
	`, s.OrganizationID, s.LocationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var id string
		var qty float64
		if err = rows.Scan(&id, &qty); err != nil {
			return nil, err
		}
		if math.Abs(qty) > 0.000001 {
			out[id] = qty
		}
	}
	return out, rows.Err()
}
func recipeUsageDelta(previous, next map[string]float64) map[string]float64 {
	ids := map[string]bool{}
	for id := range previous {
		ids[id] = true
	}
	for id := range next {
		ids[id] = true
	}
	out := map[string]float64{}
	for id := range ids {
		d := next[id] - previous[id]
		if math.Abs(d) > 0.000001 {
			out[id] = d
		}
	}
	return out
}
func applyRecipeUsageDelta(ctx context.Context, tx pgx.Tx, s scope, orderID string, delta map[string]float64) *orderPreparationError {
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
		bal, err := lockInventoryBalance(ctx, tx, s, itemID)
		if errors.Is(err, pgx.ErrNoRows) {
			return &orderPreparationError{Status: 409, Code: "recipe_item_unavailable", Message: "Un insumo de receta ya no está disponible."}
		}
		if err != nil {
			return &orderPreparationError{Status: 503, Code: "recipe_inventory_unavailable", Message: "No pudimos validar el stock de receta."}
		}
		if change > bal.Quantity+0.000001 {
			return &orderPreparationError{Status: 409, Code: "insufficient_recipe_stock", Message: "No hay stock suficiente de " + bal.Name + " para preparar el pedido."}
		}
		after := math.Round((bal.Quantity-change)*1000) / 1000
		avg := bal.AverageUnitCost
		if _, err = tx.Exec(ctx, `UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, itemID, after, avg); err != nil {
			return &orderPreparationError{Status: 503, Code: "recipe_inventory_unavailable", Message: "No pudimos actualizar el stock de receta."}
		}
		movement := "recipe_consumption"
		note := "Consumo automático por receta"
		if change < 0 {
			movement = "recipe_reversal"
			note = "Reversa de consumo por receta"
		}
		if err = insertValuedMovement(ctx, tx, s, bal.ProductID, itemID, movement, -change, after, bal.AverageUnitCost, "order", orderID, note); err != nil {
			return &orderPreparationError{Status: 503, Code: "recipe_inventory_unavailable", Message: "No pudimos registrar el Kárdex de receta."}
		}
	}
	return nil
}
func (a *API) listRecipes(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListRecipes(w, r)
}
func (a *API) getRecipe(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.GetRecipe(w, r)
}
func (a *API) saveRecipe(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.SaveRecipe(w, r)
}
