package domain

import "time"

// ValidReceiptDateFilters validates the ISO date boundaries of purchasing listings.
// Purchasing owns this invariant; it must not import Inventory to validate dates.
func ValidReceiptDateFilters(values ...string) bool {
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return false
		}
	}
	return true
}
