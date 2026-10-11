package httpapi

import "testing"

func TestRestaurantSetupCoreReady(t *testing.T) {
	tests := []struct {
		name        string
		serviceMode string
		counts      restaurantSetupCounts
		want        bool
	}{
		{name: "counter ready without tables", serviceMode: "counter", counts: restaurantSetupCounts{Categories: 1, Products: 1, CashRegisters: 1}, want: true},
		{name: "dining room requires tables", serviceMode: "dine_in", counts: restaurantSetupCounts{Categories: 1, Products: 1, CashRegisters: 1}, want: false},
		{name: "mixed ready with tables", serviceMode: "mixed", counts: restaurantSetupCounts{Categories: 1, Products: 1, CashRegisters: 1, Tables: 1}, want: true},
		{name: "catalog remains mandatory", serviceMode: "counter", counts: restaurantSetupCounts{CashRegisters: 1}, want: false},
		{name: "optional modules do not block operation", serviceMode: "counter", counts: restaurantSetupCounts{Categories: 1, Products: 1, CashRegisters: 1, InventoryItems: 0, Recipes: 0, Suppliers: 0}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := restaurantSetupCoreReady(tt.serviceMode, tt.counts); got != tt.want {
				t.Fatalf("restaurantSetupCoreReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
