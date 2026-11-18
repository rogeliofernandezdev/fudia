package domain

import (
	"math"
	"testing"
)

func TestRecipeUnitConversionsUseExactDecimalEquivalences(t *testing.T) {
	for _, tc := range []struct {
		from, to, family, sourceFactor, targetFactor string
		quantity                                     float64
		want                                         string
	}{
		{"g", "kg", "mass", "1", "1000", 200, "0.200000"},
		{"g", "kg", "mass", "1", "1000", 0.5, "0.000500"},
		{"g", "kg", "mass", "1", "1000", 0.001, "0.000001"},
		{"kg", "g", "mass", "1000", "1", 0.02, "20.000000"},
		{"ml", "l", "volume", "1", "1000", 40, "0.040000"},
		{"l", "ml", "volume", "1000", "1", 0.5, "500.000000"},
		{"und", "und", "count", "1", "1", 2, "2.000000"},
		{"custom", "custom", "custom", "1", "1", 0.25, "0.250000"},
	} {
		t.Run(tc.from+" to "+tc.to+tc.want, func(t *testing.T) {
			rule := RecipeUnitRule{InventoryUnit: tc.to, ConsumptionUnit: tc.from, InventoryFamily: tc.family, ConsumptionFamily: tc.family, InventoryFactor: tc.targetFactor, ConsumptionFactor: tc.sourceFactor}
			got, invalid := ConvertRecipeQuantity(tc.quantity, rule)
			if invalid != "" || got.BaseQuantity != tc.want || got.ConsumptionUnit != tc.from {
				t.Fatalf("conversion: %#v %s", got, invalid)
			}
		})
	}
}

func TestRecipeUnitsRejectIncompatibleAndUnrepresentableQuantities(t *testing.T) {
	rule := RecipeUnitRule{InventoryUnit: "kg", ConsumptionUnit: "g", InventoryFamily: "mass", ConsumptionFamily: "mass", InventoryFactor: "1000", ConsumptionFactor: "1"}
	for _, quantity := range []float64{0, -1, math.NaN(), math.Inf(1), 1e14, 0.0001, 0.0000001} {
		if _, invalid := ConvertRecipeQuantity(quantity, rule); invalid == "" {
			t.Fatalf("accepted invalid quantity %v", quantity)
		}
	}
	for _, invalidRule := range []RecipeUnitRule{
		{InventoryUnit: "kg", ConsumptionUnit: "ml", InventoryFamily: "mass", ConsumptionFamily: "volume"},
		{InventoryUnit: "botella", ConsumptionUnit: "und", InventoryFamily: "count", ConsumptionFamily: "count"},
		{InventoryUnit: "kg", ConsumptionUnit: "", InventoryFamily: "mass"},
		{InventoryUnit: "custom", ConsumptionUnit: "kg", InventoryFamily: "custom", ConsumptionFamily: "mass"},
	} {
		if _, invalid := ConvertRecipeQuantity(1, invalidRule); invalid == "" {
			t.Fatalf("accepted invalid unit %#v", invalidRule)
		}
	}
}
