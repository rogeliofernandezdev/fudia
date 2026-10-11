package domain

import "time"

type RecipeItemView struct {
	ID              string `json:"id"`
	InventoryItemID string `json:"inventoryItemId"`
	Name            string `json:"name"`
	Unit            string `json:"unit"`
	Quantity        string `json:"quantity"`
	WastePercent    string `json:"wastePercent"`
}
type RecipeView struct {
	ID            string           `json:"id"`
	ProductID     string           `json:"productId"`
	ProductName   string           `json:"productName"`
	YieldQuantity string           `json:"yieldQuantity"`
	Notes         string           `json:"notes"`
	Active        bool             `json:"active"`
	UpdatedAt     time.Time        `json:"updatedAt"`
	Items         []RecipeItemView `json:"items"`
}
type RecipeInput struct {
	YieldQuantity float64                 `json:"yieldQuantity"`
	Notes         string                  `json:"notes"`
	Active        bool                    `json:"active"`
	Items         []RecipeIngredientInput `json:"items"`
}
type RecipeIngredientInput struct {
	InventoryItemID string  `json:"inventoryItemId"`
	Quantity        float64 `json:"quantity"`
	WastePercent    float64 `json:"wastePercent"`
}
type RecipeSummary struct {
	ID            string    `json:"id"`
	ProductID     string    `json:"productId"`
	ProductName   string    `json:"productName"`
	YieldQuantity string    `json:"yieldQuantity"`
	Notes         string    `json:"notes"`
	Active        bool      `json:"active"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ItemCount     int       `json:"itemCount"`
}
