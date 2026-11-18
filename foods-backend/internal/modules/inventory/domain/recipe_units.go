package domain

import (
	"math"
	"math/big"
	"strconv"
)

type RecipeUnitRule struct {
	InventoryUnit, ConsumptionUnit, InventoryFamily, ConsumptionFamily, InventoryFactor, ConsumptionFactor string
}
type RecipeQuantity struct{ BaseQuantity, ConsumptionQuantity, ConsumptionUnit, Factor string }

// ConvertRecipeQuantity uses decimal rationals so a unit change cannot introduce binary rounding.
func ConvertRecipeQuantity(quantity float64, rule RecipeUnitRule) (RecipeQuantity, string) {
	var out RecipeQuantity
	if math.IsNaN(quantity) || math.IsInf(quantity, 0) || quantity <= 0 || quantity >= 1e14 {
		return out, "La cantidad debe ser mayor que cero y menor que 100000000000000."
	}
	if rule.ConsumptionUnit == "" || (rule.InventoryUnit != rule.ConsumptionUnit &&
		(rule.InventoryFamily != rule.ConsumptionFamily || (rule.InventoryFamily != "mass" && rule.InventoryFamily != "volume"))) {
		return out, "Elige una unidad de consumo compatible con la unidad de inventario."
	}
	amount, _ := new(big.Rat).SetString(strconv.FormatFloat(quantity, 'f', -1, 64))
	rounded, _ := new(big.Rat).SetString(amount.FloatString(6))
	if amount.Cmp(rounded) != 0 {
		return out, "Usa hasta seis decimales en la cantidad de consumo."
	}
	factor := big.NewRat(1, 1)
	if rule.InventoryUnit != rule.ConsumptionUnit {
		base, ok := new(big.Rat).SetString(rule.InventoryFactor)
		if !ok || base.Sign() <= 0 {
			return out, "La unidad de inventario no tiene una conversión válida."
		}
		consumed, ok := new(big.Rat).SetString(rule.ConsumptionFactor)
		if !ok || consumed.Sign() <= 0 {
			return out, "La unidad de consumo no tiene una conversión válida."
		}
		factor.Quo(consumed, base)
	}
	base := new(big.Rat).Mul(amount, factor)
	roundedBase, _ := new(big.Rat).SetString(base.FloatString(6))
	if base.Sign() <= 0 || base.Cmp(big.NewRat(100000000000, 1)) >= 0 || base.Cmp(roundedBase) != 0 {
		return out, "La equivalencia debe ser mayor que cero, menor que 100000000000 y admitir hasta seis decimales en la unidad de inventario."
	}
	return RecipeQuantity{BaseQuantity: base.FloatString(6), ConsumptionQuantity: amount.FloatString(6), ConsumptionUnit: rule.ConsumptionUnit, Factor: factor.FloatString(9)}, ""
}
