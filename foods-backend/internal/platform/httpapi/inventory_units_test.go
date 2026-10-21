package httpapi

import "testing"

func TestNormalizeInventoryUnit(t *testing.T) {
	in, invalid := normalizeInventoryUnit(inventoryUnitInput{Code: " SACO-25 ", Name: " Saco de 25 kg "})
	if invalid != "" || in.Code != "saco-25" || in.Name != "Saco de 25 kg" {
		t.Fatalf("normalization: %#v %s", in, invalid)
	}
	for _, in := range []inventoryUnitInput{{Code: "", Name: "Bolsa"}, {Code: "bolsa grande", Name: "Bolsa"}, {Code: "123", Name: "Unidad"}, {Code: "valid", Name: " "}, {Code: "abcdefghijklmnopqrstuvwxyz0123456789", Name: "Long"}} {
		if _, invalid := normalizeInventoryUnit(in); invalid == "" {
			t.Fatalf("accepted invalid unit: %#v", in)
		}
	}
}
