package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math"
	"strings"
)

func (a *Catalog) checkRecipes(ctx context.Context, s scope, write bool) error {
	recipes, inventory, err := a.repo.RecipeModules(ctx, s)
	if err != nil {
		return menuFailure(apperror.Unavailable, "module_unavailable", "No pudimos validar el módulo de recetas.")
	}
	if !recipes || !inventory {
		message := "El módulo Recetas no está habilitado para esta empresa."
		if write {
			message = "Habilita Recetas antes de configurarlas."
		}
		return menuFailure(apperror.Forbidden, "module_disabled", message)
	}
	return nil
}
func (a *Catalog) Recipes(ctx context.Context, s scope, f CatalogFilter) (RecipePage, error) {
	var out RecipePage
	if err := a.checkRecipes(ctx, s, false); err != nil {
		return out, err
	}
	out, err := a.repo.Recipes(ctx, s, f)
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "recipes_unavailable", "No pudimos cargar las recetas.")
	}
	return out, nil
}
func (a *Catalog) Recipe(ctx context.Context, s scope, productID string) (domain.RecipeView, error) {
	var out domain.RecipeView
	if err := a.checkRecipes(ctx, s, false); err != nil {
		return out, err
	}
	out, err := a.repo.Recipe(ctx, s, productID)
	if errors.Is(err, ErrNotFound) {
		return out, menuFailure(apperror.NotFound, "recipe_not_found", "El producto todavía no tiene receta.")
	}
	if err != nil {
		return out, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos cargar la receta.")
	}
	return out, nil
}
func (a *Catalog) SaveRecipe(ctx context.Context, s scope, productID string, in domain.RecipeInput) (map[string]any, error) {
	var result map[string]any
	if err := a.checkRecipes(ctx, s, true); err != nil {
		return nil, err
	}
	if math.IsNaN(in.YieldQuantity) || math.IsInf(in.YieldQuantity, 0) || in.YieldQuantity < 0.001 || in.YieldQuantity >= 1e11 || len(in.Items) == 0 || len(in.Items) > 100 || len(in.Notes) > 500 {
		return result, menuFailure(apperror.Invalid, "invalid_recipe", "Indica rendimiento e insumos válidos.")
	}
	in.YieldQuantity = math.Round(in.YieldQuantity*1000) / 1000
	seen := map[string]bool{}
	for i := range in.Items {
		x := &in.Items[i]
		x.InventoryItemID = strings.TrimSpace(x.InventoryItemID)
		x.WastePercent = math.Round(x.WastePercent*10000) / 10000
		quantity := x.Quantity
		if x.ConsumptionQuantity != nil {
			quantity = *x.ConsumptionQuantity
		}
		if (x.ConsumptionQuantity == nil) != (strings.TrimSpace(x.ConsumptionUnit) == "") || x.InventoryItemID == "" || math.IsNaN(quantity) || math.IsInf(quantity, 0) || quantity <= 0 || math.IsNaN(x.WastePercent) || math.IsInf(x.WastePercent, 0) || x.WastePercent < 0 || x.WastePercent >= 100 || seen[x.InventoryItemID] {
			return result, menuFailure(apperror.Invalid, "invalid_recipe", "Los insumos de la receta no son válidos.")
		}
		seen[x.InventoryItemID] = true
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos guardar la receta.")
	}
	defer tx.Rollback(ctx)
	var productName, control string
	err = func() error {
		var err error
		productName, control, err = a.repo.LockRecipeProduct(tx.Context(), s, productID)
		return err
	}()
	if errors.Is(err, ErrNotFound) {
		return result, menuFailure(apperror.NotFound, "product_not_found", "El producto no existe.")
	}
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos validar el producto.")
	}
	if control == "inventory" {
		return result, menuFailure(apperror.Conflict, "recipe_inventory_conflict", "Un producto controlado directamente por inventario no debe descontarse también por receta.")
	}
	if a.recipeUnits == nil {
		return result, menuFailure(apperror.Unavailable, "recipe_units_unavailable", "No pudimos validar las unidades de la receta.")
	}
	for i := range in.Items {
		x := &in.Items[i]
		quantity := x.Quantity
		if x.ConsumptionQuantity != nil {
			quantity = *x.ConsumptionQuantity
		}
		normalized, unitErr := a.recipeUnits.NormalizeRecipeQuantity(tx.Context(), s, x.InventoryItemID, quantity, x.ConsumptionUnit)
		if unitErr != nil {
			return result, unitErr
		}
		x.BaseQuantity, x.ConsumptionQuantityValue, x.ConsumptionUnit, x.ConsumptionFactor = normalized.BaseQuantity, normalized.ConsumptionQuantity, normalized.ConsumptionUnit, normalized.Factor
	}
	var recipeID string
	err = func() error {
		var err error
		recipeID, err = a.repo.SaveRecipe(tx.Context(), s, productID, in)
		return err
	}()
	if err != nil {
		return result, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos guardar la receta.")
	}
	if _, err = a.repo.DeleteRecipeIngredients(tx.Context(), s, recipeID); err != nil {
		return result, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos actualizar los insumos.")
	}
	for _, x := range in.Items {
		if _, err = a.repo.CreateRecipeIngredient(tx.Context(), s, recipeID, x); err != nil {
			return result, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos guardar un insumo.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return result, menuFailure(apperror.Unavailable, "recipe_unavailable", "No pudimos confirmar la receta.")
	}
	_ = a.audit.Record(ctx, s, "recipe.saved", "product", productID)
	return map[string]any{"id": recipeID, "productId": productID, "productName": productName}, nil

}

type RecipePage struct {
	Items    []domain.RecipeSummary `json:"items"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}
