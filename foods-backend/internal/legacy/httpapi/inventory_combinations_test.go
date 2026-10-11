package httpapi

import (
	"math"
	"testing"
)

func TestInventoryPresentationAcceptsCatalogCodesAndFractionalContents(t *testing.T) {
	for _, in := range []inventoryPresentationInput{{PresentationType: "sack", UnitsPerPresentation: 25}, {PresentationType: "custom-crate", UnitsPerPresentation: 10}, {PresentationType: "bag", UnitsPerPresentation: .5}, {PresentationType: "unit", UnitsPerPresentation: 1}} {
		if _, invalid := normalizeInventoryPresentation(in); invalid != "" {
			t.Fatalf("%#v: %s", in, invalid)
		}
	}
	for _, in := range []inventoryPresentationInput{{PresentationType: "bag", UnitsPerPresentation: 0}, {PresentationType: "unit", UnitsPerPresentation: 10}, {PresentationType: "bad code", UnitsPerPresentation: 10}, {PresentationType: "bag", UnitsPerPresentation: .0001}, {PresentationType: "bag", UnitsPerPresentation: math.Inf(1)}, {PresentationType: "bag", UnitsPerPresentation: math.NaN()}} {
		if _, invalid := normalizeInventoryPresentation(in); invalid == "" {
			t.Fatalf("accepted %#v", in)
		}
	}
}
