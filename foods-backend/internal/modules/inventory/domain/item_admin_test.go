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
	for _, want := range []string{"Quedan 2.5 kilogramos", "2 órdenes de compra", "Lo usan las recetas de Ceviche, Leche de tigre", "Quítalo de esas recetas"} {
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
	for _, want := range []string{"1 orden de compra", "Lo usa la receta de Pan en otra unidad. Cambia esa receta", "decimales", "«Caja x 0.5»"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if got := UnitChangeBlockers(ItemFacts{Unit: "kg", FractionalAnyStock: true}, UnitChangeTarget{Code: "g", Name: "Gramo", Exists: true}, nil); len(got) != 0 {
		t.Fatalf("measurable unit accepts decimals: %v", got)
	}
}

func TestQuantityLabelUsesSpanishPlural(t *testing.T) {
	for _, row := range []struct {
		quantity float64
		unit     string
		want     string
	}{{1, "kilogramo", "1 kilogramo"}, {19.01, "litro", "19.01 litros"}, {2, "unidad", "2 unidades"}, {0, "botella", "0 botellas"}} {
		if got := QuantityLabel(row.quantity, row.unit); got != row.want {
			t.Fatalf("QuantityLabel(%v,%q)=%q want %q", row.quantity, row.unit, got, row.want)
		}
	}
}

func TestUnitCorrectionWithinTheSameMeasureNeedsConversion(t *testing.T) {
	gram := UnitChangeTarget{Code: "g", Name: "Gramo", Family: "mass", Exists: true}
	used := ItemFacts{Unit: "kg", UnitName: "Kilogramo", UnitFamily: "mass", HasHistory: true}
	if got := strings.Join(UnitChangeBlockers(used, gram, nil), " "); !strings.Contains(got, "Kilogramo y gramo miden lo mismo") {
		t.Fatalf("kg→g with history must be blocked: %q", got)
	}
	fresh := used
	fresh.HasHistory = false
	if got := UnitChangeBlockers(fresh, gram, nil); len(got) != 0 {
		t.Fatalf("kg→g without history is free: %v", got)
	}
	bottle := UnitChangeTarget{Code: "botella", Name: "Botella", Family: "count", Countable: true, Exists: true}
	if got := UnitChangeBlockers(used, bottle, nil); len(got) != 0 {
		t.Fatalf("kg→botella with history is a correction: %v", got)
	}
	milliliter := UnitChangeTarget{Code: "ml", Name: "Mililitro", Family: "volume", Exists: true}
	if got := UnitChangeBlockers(ItemFacts{Unit: "l", UnitName: "Litro", UnitFamily: "volume", HasHistory: true}, milliliter, nil); len(got) != 1 {
		t.Fatalf("l→ml with history must be blocked: %v", got)
	}
}
