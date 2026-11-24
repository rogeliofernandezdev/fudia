package domain

import (
	"strings"
	"testing"
)

func TestDeactivateBlockersExplainWhatToResolve(t *testing.T) {
	if got := DeactivateBlockers(ItemFacts{}); len(got) != 0 {
		t.Fatalf("empty article must deactivate: %v", got)
	}
	got := DeactivateBlockers(ItemFacts{UnitName: "Kilogramo", QuantityHere: 2.5, OpenOrdersHere: 2, Recipes: []ItemRecipeUse{{ProductName: "Ceviche"}, {ProductName: "Leche de tigre"}}})
	joined := strings.Join(got, " ")
	for _, want := range []string{"Quedan 2.5 kilogramo", "2 órdenes de compra", "las recetas de Ceviche, Leche de tigre"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}

func TestUnitChangeBlockers(t *testing.T) {
	target := UnitChangeTarget{Code: "und", Name: "Unidad", Countable: true, Exists: true}
	if got := UnitChangeBlockers(ItemFacts{Unit: "kg"}, UnitChangeTarget{}, nil); len(got) != 1 {
		t.Fatalf("unknown unit: %v", got)
	}
	if got := UnitChangeBlockers(ItemFacts{Unit: "kg", Recipes: []ItemRecipeUse{{ProductName: "Pan", ConsumptionUnit: "kg"}}}, target, nil); len(got) != 0 {
		t.Fatalf("recipe in own unit follows the correction: %v", got)
	}
	got := UnitChangeBlockers(ItemFacts{Unit: "kg", OpenOrdersAny: 1, FractionalAnyStock: true, Recipes: []ItemRecipeUse{{ProductName: "Pan", ConsumptionUnit: "g"}}}, target, []string{"Caja x 0.5"})
	joined := strings.Join(got, " ")
	for _, want := range []string{"1 orden de compra", "receta de Pan", "decimales", "«Caja x 0.5»"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if got := UnitChangeBlockers(ItemFacts{Unit: "kg", FractionalAnyStock: true}, UnitChangeTarget{Code: "g", Name: "Gramo", Exists: true}, nil); len(got) != 0 {
		t.Fatalf("measurable unit accepts decimals: %v", got)
	}
}
