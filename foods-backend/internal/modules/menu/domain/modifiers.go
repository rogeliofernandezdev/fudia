package domain

import (
	"strconv"
	"strings"
)

type ModifierOptionInput struct {
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}
type ModifierGroupInput struct {
	Name          string                `json:"name"`
	Required      bool                  `json:"required"`
	MinSelections int                   `json:"minSelections"`
	MaxSelections int                   `json:"maxSelections"`
	Options       []ModifierOptionInput `json:"options"`
}
type ModifierConfigInput struct {
	Groups []ModifierGroupInput `json:"groups"`
}
type ModifierOptionView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}
type ModifierGroupView struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Required      bool                 `json:"required"`
	MinSelections int                  `json:"minSelections"`
	MaxSelections int                  `json:"maxSelections"`
	Options       []ModifierOptionView `json:"options"`
}
type ModifierConfigView struct {
	ProductID string              `json:"productId"`
	Groups    []ModifierGroupView `json:"groups"`
}

func ValidateModifierConfig(in ModifierConfigInput) string {
	groupNames := map[string]bool{}
	for _, group := range in.Groups {
		name := strings.TrimSpace(group.Name)
		if name == "" || len([]rune(name)) > 80 {
			return "Cada grupo necesita un nombre de hasta 80 caracteres."
		}
		key := strings.ToLower(name)
		if groupNames[key] {
			return "Los grupos de modificadores necesitan nombres distintos."
		}
		groupNames[key] = true
		if group.MinSelections < 0 || group.MaxSelections < 1 ||
			group.MinSelections > group.MaxSelections {
			return "Revisa los mínimos y máximos de cada grupo."
		}
		if group.Required && group.MinSelections < 1 {
			return "Un grupo obligatorio debe exigir al menos una selección."
		}
		if group.MaxSelections > len(group.Options) {
			return "El máximo de selecciones no puede superar las opciones disponibles."
		}

		optionNames := map[string]bool{}
		for _, option := range group.Options {
			optionName := strings.TrimSpace(option.Name)
			if optionName == "" || len([]rune(optionName)) > 80 {
				return "Cada opción necesita un nombre de hasta 80 caracteres."
			}
			optionKey := strings.ToLower(optionName)
			if optionNames[optionKey] {
				return "Las opciones de un mismo grupo necesitan nombres distintos."
			}
			optionNames[optionKey] = true
			value := strings.TrimSpace(option.Surcharge)
			if value == "" {
				value = "0"
			}
			surcharge, err := strconv.ParseFloat(value, 64)
			if err != nil || surcharge < 0 {
				return "Cada recargo debe ser un importe válido mayor o igual a cero."
			}
		}
	}
	return ""
}
