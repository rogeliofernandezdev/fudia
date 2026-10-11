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
