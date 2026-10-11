package domain

import "strings"

type Category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
	Active       bool   `json:"active"`
	ProductScope string `json:"productScope"`
	ProductCount int    `json:"productCount"`
}
type CategoryInput struct {
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
	Active       *bool  `json:"active,omitempty"`
	ProductScope string `json:"productScope"`
}
type Product struct {
	ID                 string   `json:"id"`
	SKU                string   `json:"sku"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	CategoryID         *string  `json:"categoryId"`
	CategoryName       *string  `json:"categoryName"`
	Price              string   `json:"price"`
	Active             bool     `json:"active"`
	ServiceDestination string   `json:"serviceDestination"`
	ProductType        string   `json:"productType"`
	QuantityControl    string   `json:"quantityControl"`
	ImageURL           *string  `json:"imageUrl"`
	PrepMinutes        *int     `json:"prepMinutes"`
	Allergens          []string `json:"allergens"`
	Featured           bool     `json:"featured"`
	CostPrice          *string  `json:"costPrice"`
	AvailableFrom      *string  `json:"availableFrom"`
	AvailableUntil     *string  `json:"availableUntil"`
	AvailableDays      []int    `json:"availableDays"`
	AvailableUntilTime *string  `json:"availableUntilTime"`
}
type ProductInput struct {
	InitialPortionQuantity *int     `json:"initialPortionQuantity,omitempty"`
	SKU                    string   `json:"sku"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	CategoryID             *string  `json:"categoryId"`
	Price                  string   `json:"price"`
	Active                 *bool    `json:"active,omitempty"`
	ServiceDestination     string   `json:"serviceDestination"`
	ProductType            string   `json:"productType"`
	QuantityControl        string   `json:"quantityControl"`
	ImageURL               *string  `json:"imageUrl"`
	PrepMinutes            *int     `json:"prepMinutes"`
	Allergens              []string `json:"allergens"`
	Featured               *bool    `json:"featured,omitempty"`
	CostPrice              *string  `json:"costPrice"`
	AvailableFrom          *string  `json:"availableFrom"`
	AvailableUntil         *string  `json:"availableUntil"`
	AvailableDays          []int    `json:"availableDays"`
	AvailableUntilTime     *string  `json:"availableUntilTime"`
}
type ProductListItem struct {
	Product
	AvailableQuantity *float64 `json:"availableQuantity"`
	InventoryUnit     *string  `json:"inventoryUnit"`
}

func NormalizeCategoryProductScope(value, fallback string) (string, string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback, ""
	}
	switch value {
	case "prepared", "retail", "both":
		return value, ""
	default:
		return value, "El uso de la categoría no es válido."
	}
}
func NormalizeProduct(in ProductInput) (ProductInput, string) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Price) == "" {
		return in, "Nombre y precio son obligatorios."
	}
	in.ProductType = strings.ToLower(strings.TrimSpace(in.ProductType))
	if in.ProductType == "" {
		in.ProductType = "prepared"
	}
	switch in.ProductType {
	case "prepared", "retail":
	default:
		return in, "El tipo de producto no es válido."
	}
	if in.ServiceDestination == "" {
		in.ServiceDestination = "kitchen"
		if in.ProductType == "retail" {
			in.ServiceDestination = "direct"
		}
	}
	if in.ServiceDestination != "kitchen" && in.ServiceDestination != "bar" && in.ServiceDestination != "direct" {
		return in, "Selecciona Cocina, Barra o Entrega directa."
	}
	if in.QuantityControl == "" {
		in.QuantityControl = "none"
	}
	switch in.QuantityControl {
	case "none", "portions", "inventory":
	default:
		return in, "El control de cantidad no es válido."
	}
	return in, ""
}
func ValidateInitialPortions(in ProductInput) string {
	if in.QuantityControl == "portions" {
		if in.InitialPortionQuantity == nil || *in.InitialPortionQuantity < 1 || *in.InitialPortionQuantity > 2147483647 {
			return "Ingresa una cantidad disponible hoy entera entre 1 y 2147483647."
		}
	} else if in.InitialPortionQuantity != nil {
		return "La cantidad disponible hoy solo corresponde a Porciones preparadas."
	}
	return ""
}

var ServiceDestinations = []map[string]string{{"value": "kitchen", "label": "Cocina"}, {"value": "bar", "label": "Barra"}, {"value": "direct", "label": "Entrega directa"}}
