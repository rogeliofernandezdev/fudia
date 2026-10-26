package domain

import (
	"regexp"
	"strings"
)

type Category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
	Active       bool   `json:"active"`
	ProductScope string `json:"productScope"`
	ProductCount int    `json:"productCount"`
}
// AllergenInput registers an organization allergen; duplicates are checked
// against the shared catalog and the organization's own entries.
type AllergenInput struct {
	Name string `json:"name"`
}
type Allergen struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CategoryInput struct {
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
	Active       *bool  `json:"active,omitempty"`
	ProductScope string `json:"productScope"`
}
type Product struct {
	ImageView
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
	// HasRecipe indica receta activa: el formulario avisa si un producto con receta se marca «Sin preparación».
	HasRecipe bool `json:"hasRecipe"`
}
type ProductInput struct {
	ImageID                *string  `json:"imageId,omitempty"`
	RemoveImage            bool     `json:"removeImage,omitempty"`
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
		return in, "Selecciona Cocina, Barra o Sin preparación."
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
var categoryIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// NormalizeProductCategory exige una categoría al crear o editar un producto:
// «Sin categoría» ya no es una opción. La existencia y el estado de la
// categoría se comprueban después contra la empresa.
func NormalizeProductCategory(in ProductInput) (ProductInput, string) {
	if in.CategoryID == nil || strings.TrimSpace(*in.CategoryID) == "" {
		in.CategoryID = nil
		return in, "Selecciona una categoría para el producto."
	}
	categoryID := strings.TrimSpace(*in.CategoryID)
	in.CategoryID = &categoryID
	if !categoryIDPattern.MatchString(categoryID) {
		return in, "La categoría seleccionada no es válida."
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

// ServiceDestinations es el catálogo único de destinos de atención. «label» se usa al elegir el destino del
// producto; «shortLabel», en las líneas del pedido. «direct» no pasa por Cocina ni Barra: entra como Listo
// y lo entrega el mozo (gaseosas, agua, cerveza en botella, postres ya porcionados).
var ServiceDestinations = []map[string]string{
	{"value": "kitchen", "label": "Cocina", "shortLabel": "Cocina"},
	{"value": "bar", "label": "Barra", "shortLabel": "Barra"},
	{"value": "direct", "label": "Sin preparación (lo entrega el mozo)", "shortLabel": "Sin preparación"},
}
