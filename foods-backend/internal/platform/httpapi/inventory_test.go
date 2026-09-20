package httpapi

import "testing"

func TestNormalizeInventoryEntryRequiresSingleProductSource(t *testing.T) {
	valid := inventoryEntryInput{ProductID: "product-1", Quantity: 5, Unit: "und"}
	if _, invalid := normalizeInventoryEntry(valid); invalid != "" {
		t.Fatalf("expected existing product entry to be valid: %s", invalid)
	}
	both := valid
	both.NewProduct = &inventoryNewProductInput{Name: "Agua", Price: "3.00"}
	if _, invalid := normalizeInventoryEntry(both); invalid == "" {
		t.Fatal("expected productId + newProduct to be rejected")
	}
	if _, invalid := normalizeInventoryEntry(inventoryEntryInput{Quantity: 5}); invalid == "" {
		t.Fatal("expected entry without product source to be rejected")
	}
}

func TestNormalizeInventoryEntryDefaultsUnitAndRequiresPositiveQuantity(t *testing.T) {
	in, invalid := normalizeInventoryEntry(inventoryEntryInput{ProductID: "product-1", Quantity: 2})
	if invalid != "" {
		t.Fatalf("unexpected error: %s", invalid)
	}
	if in.Unit != "und" {
		t.Fatalf("expected default unit und, got %q", in.Unit)
	}
	if _, invalid := normalizeInventoryEntry(inventoryEntryInput{ProductID: "product-1", Quantity: 0}); invalid == "" {
		t.Fatal("expected zero quantity to be rejected")
	}
}

func TestPreparedQuantityUsageUsesProductIds(t *testing.T) {
	productID := "product-direct"
	items := []preparedOrderItem{
		{ProductID: &productID, ItemType: "product", Qty: 2},
		{ItemType: "combo", Qty: 3, Selections: []preparedOrderSelection{{ProductID: "drink"}, {ProductID: "side"}}},
	}
	usage := preparedQuantityUsage(items)
	if usage["product-direct"] != 2 || usage["drink"] != 3 || usage["side"] != 3 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}

func TestQuantityUsageDeltaSupportsSaleAndReversal(t *testing.T) {
	delta := quantityUsageDelta(map[string]float64{"water": 3, "soup": 2}, map[string]float64{"water": 1, "cola": 4})
	if delta["water"] != -2 || delta["soup"] != -2 || delta["cola"] != 4 {
		t.Fatalf("unexpected delta: %#v", delta)
	}
}
