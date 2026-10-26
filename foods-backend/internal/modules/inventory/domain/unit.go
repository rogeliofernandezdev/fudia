package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

type Unit struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	// Countable indica que la unidad se cuenta por piezas (botella, lata...).
	Countable bool `json:"countable"`
}

// PresentationRule es lo que la unidad base de un artículo permite al guardar
// una presentación.
type PresentationRule struct {
	Allowed   bool
	Countable bool
	UnitName  string
}

// PresentationContentError rechaza presentaciones de unidades contables con
// menos de una pieza; las unidades de medida admiten fracciones.
func PresentationContentError(rule PresentationRule, presentationType string, unitsPerPresentation float64) string {
	if !rule.Countable || presentationType == "unit" || unitsPerPresentation >= 1 {
		return ""
	}
	unit := strings.ToLower(strings.TrimSpace(rule.UnitName))
	if unit == "" {
		unit = "unidad"
	}
	return "La presentación debe contener al menos 1 " + unit + ". Las fracciones solo se usan con unidades de medida como kilos o litros."
}
type UnitInput struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var inventoryUnitCode = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

func NormalizeUnit(in UnitInput) (UnitInput, string) {
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if !inventoryUnitCode.MatchString(in.Code) {
		return in, "El código debe tener hasta 32 caracteres: letras, números, guion o guion bajo."
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 80 {
		return in, "Ingresa un nombre de hasta 80 caracteres."
	}
	return in, ""
}

func ValidUnitCode(code string) bool { return inventoryUnitCode.MatchString(code) }
