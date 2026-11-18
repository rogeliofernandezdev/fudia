package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"strings"
)

type RecipeUnitRepository interface {
	RecipeUnitRule(context.Context, scope, string, string) (domain.RecipeUnitRule, error)
}
type RecipeUnits struct{ repo RecipeUnitRepository }

func NewRecipeUnits(repo RecipeUnitRepository) *RecipeUnits { return &RecipeUnits{repo: repo} }
func (a *RecipeUnits) NormalizeRecipeQuantity(ctx context.Context, s scope, itemID string, quantity float64, unit string) (contracts.RecipeQuantity, error) {
	rule, err := a.repo.RecipeUnitRule(ctx, s, itemID, strings.TrimSpace(unit))
	if errors.Is(err, ErrNotFound) {
		return contracts.RecipeQuantity{}, stockFailure(apperror.Conflict, "recipe_item_invalid", "Un insumo ya no está disponible.")
	}
	if err != nil {
		return contracts.RecipeQuantity{}, stockFailure(apperror.Unavailable, "recipe_units_unavailable", "No pudimos validar las unidades del insumo.")
	}
	out, invalid := domain.ConvertRecipeQuantity(quantity, rule)
	if invalid != "" {
		return contracts.RecipeQuantity{}, stockFailure(apperror.Invalid, "invalid_recipe_unit", invalid)
	}
	return contracts.RecipeQuantity{BaseQuantity: out.BaseQuantity, ConsumptionQuantity: out.ConsumptionQuantity, ConsumptionUnit: out.ConsumptionUnit, Factor: out.Factor}, nil
}
