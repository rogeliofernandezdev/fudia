package domain

type PublicMenuItem struct {
	ImageView
	ProductID    string  `json:"productId"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Price        string  `json:"price"`
	CategoryName *string `json:"categoryName"`
	ImageURL     *string `json:"imageUrl"`
	Status       string  `json:"status"`
	IsCombo      bool    `json:"isCombo"`
	HasModifiers bool    `json:"hasModifiers"`
}
type PublicComboOption struct {
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
	Available bool   `json:"available"`
}
type PublicComboGroup struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Required      bool                `json:"required"`
	MinSelections int                 `json:"minSelections"`
	MaxSelections int                 `json:"maxSelections"`
	Options       []PublicComboOption `json:"options"`
}
type PublicComboDetail struct {
	ImageView
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Price       string             `json:"price"`
	ImageURL    *string            `json:"imageUrl"`
	Groups      []PublicComboGroup `json:"groups"`
}
type PublicMenuSnapshot struct {
	Item         PublicMenuItem
	Availability AvailabilitySnapshot
}
