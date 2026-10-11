package domain

type AvailabilityItem struct {
	ProductID       string   `json:"productId"`
	Name            string   `json:"name"`
	Price           string   `json:"price"`
	CategoryID      *string  `json:"categoryId"`
	CategoryName    *string  `json:"categoryName"`
	ImageURL        *string  `json:"imageUrl"`
	QuantityControl string   `json:"quantityControl"`
	Status          string   `json:"status"`
	Source          string   `json:"source"`
	ManualStatus    string   `json:"manualStatus"`
	PortionQuantity *int     `json:"portionQuantity"`
	SoldQuantity    int      `json:"soldQuantity"`
	Remaining       *float64 `json:"remaining"`
	InventoryUnit   *string  `json:"inventoryUnit"`
	Note            string   `json:"note"`
	BusinessDate    string   `json:"businessDate"`
}
type AvailabilityInput struct {
	Status          string `json:"status"`
	PortionQuantity *int   `json:"portionQuantity"`
	Note            string `json:"note"`
	Reason          string `json:"reason"`
}
type AvailabilityHistoryItem struct {
	ID                      string  `json:"id"`
	CreatedAt               string  `json:"createdAt"`
	BusinessDate            *string `json:"businessDate"`
	UserID                  *string `json:"userId"`
	UserName                string  `json:"userName"`
	Reason                  *string `json:"reason"`
	PreviousPortionQuantity *int    `json:"previousPortionQuantity"`
	PortionQuantity         *int    `json:"portionQuantity"`
	SoldQuantity            *int    `json:"soldQuantity"`
	PreviousManualStatus    *string `json:"previousManualStatus"`
	ManualStatus            *string `json:"manualStatus"`
}
type AvailabilitySnapshot struct {
	Item                            AvailabilityItem
	ManualStatus                    string
	ScheduleAvailable, IsCombo      bool
	InventoryQuantity, MinimumStock float64
}

func ResolveAvailability(raw AvailabilitySnapshot, day string) AvailabilityItem {
	item := raw.Item
	item.BusinessDate = day
	if raw.ManualStatus == "low" {
		raw.ManualStatus = "available"
	}
	item.ManualStatus = raw.ManualStatus
	item.Status, item.Source = "available", "always"

	if raw.ManualStatus == "sold_out" {
		item.Status, item.Source = "sold_out", "manual_override"
	} else if !raw.ScheduleAvailable {
		item.Status, item.Source = "unavailable", "schedule"
	} else {
		switch item.QuantityControl {
		case "portions":
			item.Source = "portions"
			remaining := 0.0
			if item.PortionQuantity != nil {
				remaining = float64(*item.PortionQuantity - item.SoldQuantity)
				if remaining < 0 {
					remaining = 0
				}
			}
			item.Remaining = &remaining
			if item.PortionQuantity == nil || remaining <= 0 {
				item.Status = "sold_out"
			} else if remaining <= 3 {
				item.Status = "low"
			}
		case "inventory":
			item.Source = "inventory"
			remaining := raw.InventoryQuantity
			item.Remaining = &remaining
			if raw.InventoryQuantity <= 0 {
				item.Status = "sold_out"
			} else if (raw.MinimumStock > 0 && raw.InventoryQuantity <= raw.MinimumStock) || raw.InventoryQuantity <= 3 {
				item.Status = "low"
			}
		}
	}

	return item
}
