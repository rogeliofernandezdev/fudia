package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// RecipeQuantity preserves the entered unit and its validated equivalence to physical stock.
type RecipeQuantity struct{ BaseQuantity, ConsumptionQuantity, ConsumptionUnit, Factor string }
type RecipeUnitResolver interface {
	NormalizeRecipeQuantity(context.Context, requestscope.Scope, string, float64, string) (RecipeQuantity, error)
}
