package domain

import "strings"

type ComboOptionInput struct {
	ProductID string `json:"productId"`
	Surcharge string `json:"surcharge"`
	Quota     *int   `json:"quota"`
}
type ComboGroupInput struct {
	Name          string             `json:"name"`
	Required      bool               `json:"required"`
	MinSelections int                `json:"minSelections"`
	MaxSelections int                `json:"maxSelections"`
	Options       []ComboOptionInput `json:"options"`
}
type ComboInput struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Price          string            `json:"price"`
	CategoryID     *string           `json:"categoryId"`
	AvailableFrom  *string           `json:"availableFrom"`
	AvailableUntil *string           `json:"availableUntil"`
	AvailableDays  []int             `json:"availableDays"`
	Groups         []ComboGroupInput `json:"groups"`
}

func ValidateCombo(in ComboInput) string {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Price) == "" {
		return "Nombre y precio son obligatorios."
	}
	if len(in.Groups) == 0 {
		return "Agrega al menos un grupo al menú."
	}
	groupNames := map[string]bool{}
	for _, group := range in.Groups {
		for _, option := range group.Options {
			if option.Quota != nil && *option.Quota <= 0 {
				return "La cantidad reservada debe ser mayor a cero."
			}
		}
		groupName := strings.TrimSpace(group.Name)
		if groupName == "" || group.MaxSelections < 1 || group.MinSelections < 0 || group.MaxSelections < group.MinSelections {
			return "Revisa las reglas de selección de cada grupo."
		}
		groupKey := strings.ToLower(groupName)
		if groupNames[groupKey] {
			return "Cada grupo del menú necesita un nombre único."
		}
		groupNames[groupKey] = true
		if group.Required && len(group.Options) < group.MinSelections {
			return "Cada grupo obligatorio necesita alternativas suficientes."
		}
	}
	return ""
}
func NormalizedSurcharge(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0"
	}
	return value
}

type ComboSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       string `json:"price"`
	Active      bool   `json:"active"`
	GroupCount  int    `json:"groupCount"`
	OptionCount int    `json:"optionCount"`
	// OutsideDates is true when "now" is outside the menu's sale dates (expired or not started), so it is not sold
	// even though it is active.
	OutsideDates bool `json:"outsideDates"`
}
type OrderComboSummary struct {
	ImageView
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       string  `json:"price"`
	ImageURL    *string `json:"imageUrl"`
	GroupCount  int     `json:"groupCount"`
	OptionCount int     `json:"optionCount"`
}
type ComboOption struct {
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
	Quota     *int   `json:"quota"`
}
type OrderComboOption struct {
	ProductID      string   `json:"productId"`
	Name           string   `json:"name"`
	Surcharge      string   `json:"surcharge"`
	Available      bool     `json:"available"`
	Remaining      *float64 `json:"remaining"`
	QuotaRemaining *float64 `json:"quotaRemaining"`
}
type ComboGroup struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Required      bool          `json:"required"`
	MinSelections int           `json:"minSelections"`
	MaxSelections int           `json:"maxSelections"`
	Options       []ComboOption `json:"options"`
}
type OrderComboGroup struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Required      bool               `json:"required"`
	MinSelections int                `json:"minSelections"`
	MaxSelections int                `json:"maxSelections"`
	Options       []OrderComboOption `json:"options"`
}
type Combo struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Price          string       `json:"price"`
	Active         bool         `json:"active"`
	AvailableFrom  *string      `json:"availableFrom"`
	AvailableUntil *string      `json:"availableUntil"`
	AvailableDays  []int        `json:"availableDays"`
	Groups         []ComboGroup `json:"groups"`
}
type OrderCombo struct {
	ImageView
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Price       string            `json:"price"`
	ImageURL    *string           `json:"imageUrl"`
	Groups      []OrderComboGroup `json:"groups"`
}
