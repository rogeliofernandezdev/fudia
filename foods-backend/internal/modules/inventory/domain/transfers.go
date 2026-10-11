package domain

import "time"

type TransferInput struct {
	IdempotencyKey string         `json:"idempotencyKey"`
	ToLocationID   string         `json:"toLocationId"`
	Notes          string         `json:"notes"`
	Items          []TransferLine `json:"items"`
}
type TransferLine struct {
	InventoryItemID string  `json:"inventoryItemId"`
	Quantity        float64 `json:"quantity"`
}

type LocationBalance struct{ Quantity, Cost float64 }
type TransferSummary struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	FromLocationName string    `json:"fromLocationName"`
	ToLocationName   string    `json:"toLocationName"`
	Notes            string    `json:"notes"`
	CreatedByName    string    `json:"createdByName"`
	CreatedAt        time.Time `json:"createdAt"`
	ItemCount        int       `json:"itemCount"`
}
