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
	if in.PresentationType != "unit" || in.UnitsPerPresentation != 1 {
		t.Fatalf("expected base unit presentation, got type=%q factor=%v", in.PresentationType, in.UnitsPerPresentation)
	}
	if _, invalid := normalizeInventoryEntry(inventoryEntryInput{ProductID: "product-1", Quantity: 0}); invalid == "" {
		t.Fatal("expected zero quantity to be rejected")
	}
}

func TestNormalizeInventoryEntryValidatesPackageConversion(t *testing.T) {
	in, invalid := normalizeInventoryEntry(inventoryEntryInput{
		ProductID: "product-1", Quantity: 5, Unit: "botella",
		PresentationType: "package", UnitsPerPresentation: 12,
	})
	if invalid != "" {
		t.Fatalf("expected package entry to be valid: %s", invalid)
	}
	if in.PresentationType != "package" || in.UnitsPerPresentation != 12 {
		t.Fatalf("unexpected package normalization: %#v", in)
	}
	if _, invalid := normalizeInventoryEntry(inventoryEntryInput{
		ProductID: "product-1", Quantity: 5, PresentationType: "package", UnitsPerPresentation: 1,
	}); invalid == "" {
		t.Fatal("expected package factor <= 1 to be rejected")
	}
	if _, invalid := normalizeInventoryEntry(inventoryEntryInput{
		ProductID: "product-1", Quantity: 1.5, PresentationType: "box", UnitsPerPresentation: 12,
	}); invalid == "" {
		t.Fatal("expected fractional package quantity to be rejected")
	}
	if _, invalid := normalizeInventoryEntry(inventoryEntryInput{
		ProductID: "product-1", Quantity: 5, PresentationType: "pallet", UnitsPerPresentation: 20,
	}); invalid == "" {
		t.Fatal("expected unsupported presentation type to be rejected")
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


func TestNormalizeInventoryEntryRequiresRetailCategory(t *testing.T) {
	input := inventoryEntryInput{
		NewProduct: &inventoryNewProductInput{Name: "Agua", Price: "3.00"},
		Quantity: 1,
		Unit: "botella",
	}
	if _, invalid := normalizeInventoryEntry(input); invalid == "" {
		t.Fatal("expected category to be required for a new vendible product")
	}
	categoryID := "category-1"
	input.NewProduct.CategoryID = &categoryID
	if _, invalid := normalizeInventoryEntry(input); invalid != "" {
		t.Fatalf("expected category to make entry valid, got %q", invalid)
	}
}
