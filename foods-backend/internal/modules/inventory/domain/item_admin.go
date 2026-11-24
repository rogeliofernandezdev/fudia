package domain

import (
	"math"
	"strconv"
	"strings"
)

// ItemLocationUse is another location that has the article in its list.
type ItemLocationUse struct {
	LocationID string `json:"locationId"`
	Name       string `json:"name"`
	Active     bool   `json:"active"`
	Quantity   string `json:"quantity"`
}

// ItemPresentation is a purchase presentation of the article with its usage flag.
type ItemPresentation struct {
	ID                   string `json:"id"`
	PresentationType     string `json:"presentationType"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
	Name                 string `json:"name"`
	IsDefault            bool   `json:"isDefault"`
	IsBase               bool   `json:"isBase"`
	Used                 bool   `json:"used"`
}

// ItemRecipeUse is an active recipe that consumes the article.
type ItemRecipeUse struct {
	ProductName     string `json:"productName"`
	ConsumptionUnit string `json:"consumptionUnit"`
}

// ItemFacts is everything the rules need to know about one article, read in
// one place so the detail screen and the commands apply the same rules.
type ItemFacts struct {
	InventoryItemID    string
	ProductID          *string
	ProductName        *string
	Name               string
	Unit               string
	UnitName           string
	UnitFamily         string
	UnitCountable      bool
	AssignedHere       bool
	ActiveHere         bool
	QuantityHere       float64
	HasHistory         bool
	OpenOrdersHere     int
	OpenOrdersAny      int
	Recipes            []ItemRecipeUse
	OtherLocations     []ItemLocationUse
	FractionalAnyStock bool
}

// ItemDetail is the management view of an article for the active location.
type ItemDetail struct {
	InventoryItemID    string             `json:"inventoryItemId"`
	ProductID          *string            `json:"productId"`
	ProductName        *string            `json:"productName"`
	Name               string             `json:"name"`
	Unit               string             `json:"unit"`
	UnitName           string             `json:"unitName"`
	UnitCountable      bool               `json:"unitCountable"`
	Active             bool               `json:"active"`
	Quantity           string             `json:"quantity"`
	MinimumStock       string             `json:"minimumStock"`
	OptimalStock       string             `json:"optimalStock"`
	HasHistory         bool               `json:"hasHistory"`
	Recipes            []ItemRecipeUse    `json:"recipes"`
	OtherLocations     []ItemLocationUse  `json:"otherLocations"`
	Presentations      []ItemPresentation `json:"presentations"`
	CanDeactivate      bool               `json:"canDeactivate"`
	DeactivateBlockers []string           `json:"deactivateBlockers"`
}

// UnitChangeTarget describes the unit the article would move to.
type UnitChangeTarget struct {
	Code      string
	Name      string
	Family    string
	Countable bool
	Exists    bool
}

// UnitChangePreview tells the screen what a unit change would do before saving.
type UnitChangePreview struct {
	Unit     string   `json:"unit"`
	UnitName string   `json:"unitName"`
	Mode     string   `json:"mode"`
	Allowed  bool     `json:"allowed"`
	Blockers []string `json:"blockers"`
	Quantity string   `json:"quantity"`
	FromUnit string   `json:"fromUnit"`
	ToUnit   string   `json:"toUnit"`
}

// ItemStatusInput activates or deactivates the article in the active location.
type ItemStatusInput struct {
	Active *bool `json:"active"`
}

// AssignableItem is a company article not active in the current location.
type AssignableItem struct {
	InventoryItemID string   `json:"inventoryItemId"`
	ProductID       *string  `json:"productId"`
	Name            string   `json:"name"`
	Unit            string   `json:"unit"`
	UnitName        string   `json:"unitName"`
	InactiveHere    bool     `json:"inactiveHere"`
	Locations       []string `json:"locations"`
}

// FormatQuantity prints a stock quantity without trailing zeros.
func FormatQuantity(value float64) string {
	return strconv.FormatFloat(math.Round(value*1e6)/1e6, 'f', -1, 64)
}

// DeactivateBlockers lists, in plain language, what prevents hiding the article
// in the active location. Stock must be zero, nothing may be pending to receive
// and no active recipe may consume it.
func DeactivateBlockers(f ItemFacts) []string {
	blockers := []string{}
	if f.QuantityHere != 0 {
		blockers = append(blockers, "Quedan "+QuantityLabel(f.QuantityHere, unitLabel(f))+" en este local. Registra un ajuste de salida para dejarlo en 0.")
	}
	if f.OpenOrdersHere > 0 {
		blockers = append(blockers, "Está en "+ordersLabel(f.OpenOrdersHere)+" de este local sin recibir. Recíbela o cancélala primero.")
	}
	if len(f.Recipes) > 0 {
		blockers = append(blockers, recipeList(f.Recipes)+". Quítalo de "+recipeNoun(f.Recipes)+" primero.")
	}
	return blockers
}

// UnitChangeBlockers lists what prevents changing the base unit. Without history
// the change is free; with history it is a correction: quantities keep their
// numbers and are read in the new unit, so recipes in another unit, pending
// purchases and fractional stock of a countable unit must be resolved first.
func UnitChangeBlockers(f ItemFacts, to UnitChangeTarget, incompatible []string) []string {
	blockers := []string{}
	if !to.Exists {
		return append(blockers, "Selecciona una unidad registrada en tu empresa.")
	}
	if f.OpenOrdersAny > 0 {
		blockers = append(blockers, "Está en "+ordersLabel(f.OpenOrdersAny)+" sin recibir. Recíbela o cancélala primero.")
	}
	recipes := []ItemRecipeUse{}
	for _, r := range f.Recipes {
		if r.ConsumptionUnit != "" && r.ConsumptionUnit != f.Unit {
			recipes = append(recipes, r)
		}
	}
	if len(recipes) > 0 {
		blockers = append(blockers, recipeList(recipes)+" en otra unidad. Cambia "+recipeNoun(recipes)+" primero.")
	}
	if to.Countable && f.FractionalAnyStock {
		blockers = append(blockers, "Hay existencias o niveles de stock con decimales. "+to.Name+" solo admite cantidades enteras: ajústalos primero.")
	}
	for _, name := range incompatible {
		blockers = append(blockers, "La presentación «"+name+"» no aplica a "+strings.ToLower(to.Name)+". Quítala primero.")
	}
	return blockers
}

func unitLabel(f ItemFacts) string {
	if f.UnitName != "" {
		return strings.ToLower(f.UnitName)
	}
	return f.Unit
}

func ordersLabel(n int) string {
	if n == 1 {
		return "1 orden de compra"
	}
	return strconv.Itoa(n) + " órdenes de compra"
}

// recipeList starts the sentence with the verb agreeing with the count:
// «Lo usa la receta de Pan» or «Lo usan las recetas de Pan, Ceviche».
func recipeList(recipes []ItemRecipeUse) string {
	names := make([]string, 0, len(recipes))
	for _, r := range recipes {
		names = append(names, r.ProductName)
	}
	if len(names) == 1 {
		return "Lo usa la receta de " + names[0]
	}
	return "Lo usan las recetas de " + strings.Join(names, ", ")
}

func recipeNoun(recipes []ItemRecipeUse) string {
	if len(recipes) == 1 {
		return "esa receta"
	}
	return "esas recetas"
}

// QuantityLabel joins a quantity with its unit in Spanish, plural unless it is 1:
// «1 kilogramo», «19.01 litros», «2 unidades».
func QuantityLabel(quantity float64, unit string) string {
	text := FormatQuantity(quantity)
	if unit == "" || text == "1" {
		return strings.TrimSpace(text + " " + unit)
	}
	return text + " " + PluralUnit(unit)
}

// PluralUnit pluralizes a lowercase unit name: litro→litros, unidad→unidades.
func PluralUnit(unit string) string {
	switch {
	case unit == "":
		return unit
	case strings.HasSuffix(unit, "z"):
		return strings.TrimSuffix(unit, "z") + "ces"
	case strings.ContainsAny(unit[len(unit)-1:], "aeiou") || strings.HasSuffix(unit, "á") || strings.HasSuffix(unit, "é") || strings.HasSuffix(unit, "ó"):
		return unit + "s"
	default:
		return unit + "es"
	}
}
