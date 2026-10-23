package domain

import (
	"strings"
	"testing"
)

func TestAdjustmentQuantityAndCostLimits(t *testing.T) {
	base := InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "entry", Reason: "surplus_adjustment", Quantity: 1}
	tooMuch := base
	tooMuch.Quantity = MaxStockQuantity
	if _, invalid := NormalizeInventoryAdjustment(tooMuch); invalid == "" {
		t.Fatal("quantity past numeric(14,3) must be rejected")
	}
	negativeCost := base
	value := -1.0
	negativeCost.UnitCost = &value
	if _, invalid := NormalizeInventoryAdjustment(negativeCost); invalid == "" {
		t.Fatal("negative cost must be rejected")
	}
	exit := base
	exit.MovementType, exit.Reason, exit.UnitCost = "exit", "waste", &value
	if out, invalid := NormalizeInventoryAdjustment(exit); invalid != "" || out.UnitCost != nil {
		t.Fatalf("exit ignores cost: %q %#v", invalid, out.UnitCost)
	}
}

func TestAdjustmentObservationCountsCharacters(t *testing.T) {
	in := InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "exit", Reason: "waste", Quantity: 1, Observation: strings.Repeat("ñ", 240)}
	if _, invalid := NormalizeInventoryAdjustment(in); invalid != "" {
		t.Fatalf("240 accented characters fit the limit: %q", invalid)
	}
}

func TestEntryMinimumIsOptional(t *testing.T) {
	in := InventoryEntryInput{InventoryItemID: "item-1", Quantity: 1}
	out, invalid := NormalizeInventoryEntry(in)
	if invalid != "" || out.MinimumStock != nil || out.MinimumStockValue() != 0 {
		t.Fatalf("absent minimum stays absent: %q %#v", invalid, out.MinimumStock)
	}
	negative := -1.0
	in.MinimumStock = &negative
	if _, invalid = NormalizeInventoryEntry(in); invalid == "" {
		t.Fatal("negative minimum must be rejected")
	}
}

func TestLocationSettingsLimits(t *testing.T) {
	if !ValidLocationSettings(LocationSettings{MinimumStock: 1, ReorderPoint: 2, OptimalStock: 3}) {
		t.Fatal("regular levels are valid")
	}
	if ValidLocationSettings(LocationSettings{OptimalStock: MaxStockQuantity}) || ValidLocationSettings(LocationSettings{MinimumStock: -1}) {
		t.Fatal("levels must be within 0 and the column limit")
	}
}
