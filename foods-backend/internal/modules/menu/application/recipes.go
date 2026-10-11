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
	if in.YieldQuantity <= 0 || len(in.Items) == 0 || len(in.Notes) > 500 {
		return result, menuFailure(apperror.Invalid, "invalid_recipe", "Indica rendimiento e insumos válidos.")
	}
	in.YieldQuantity = math.Round(in.YieldQuantity*1000) / 1000
	seen := map[string]bool{}
	for i := range in.Items {
		x := &in.Items[i]
		x.InventoryItemID = strings.TrimSpace(x.InventoryItemID)
		x.Quantity = math.Round(x.Quantity*1000) / 1000
		x.WastePercent = math.Round(x.WastePercent*10000) / 10000
		if x.InventoryItemID == "" || x.Quantity <= 0 || x.WastePercent < 0 || x.WastePercent >= 100 || seen[x.InventoryItemID] {
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
	for _, x := range in.Items {
		var ok bool
		if err = func() error { var err error; ok, err = a.repo.RecipeIngredientActive(tx.Context(), s, x); return err }(); err != nil || !ok {
			return result, menuFailure(apperror.Conflict, "recipe_item_invalid", "Un insumo ya no está disponible.")
		}
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
