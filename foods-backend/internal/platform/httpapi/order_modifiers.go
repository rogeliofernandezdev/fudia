package httpapi

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

type orderItemModifierInput struct {
	GroupID  string `json:"groupId"`
	OptionID string `json:"optionId"`
}

type orderItemModifier struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
	OptionID  string `json:"optionId"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}

type preparedOrderModifier struct {
	GroupID   string
	GroupName string
	OptionID  string
	Name      string
	Surcharge float64
}

func prepareProductModifiers(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
	productID string,
	inputs []orderItemModifierInput,
) ([]preparedOrderModifier, float64, *orderPreparationError) {
	type option struct {
		ID        string
		Name      string
		Surcharge float64
	}
	type group struct {
		ID       string
		Name     string
		Required bool
		Min      int
		Max      int
		Options  map[string]option
	}

	rows, err := tx.Query(ctx, `
		SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
		       COALESCE(o.id::text,''),COALESCE(o.name,''),COALESCE(o.surcharge::float8,0)
		FROM product_modifier_groups g
		LEFT JOIN product_modifier_options o
		  ON o.group_id=g.id AND o.organization_id=g.organization_id
		WHERE g.organization_id=$1 AND g.product_id=$2
		ORDER BY g.sort_order,g.id,o.sort_order,o.id
	`, organizationID, productID)
	if err != nil {
		return nil, 0, &orderPreparationError{Status: 503, Code: "modifiers_unavailable", Message: "No pudimos validar los modificadores del producto."}
	}
	defer rows.Close()

	groups := map[string]*group{}
	groupOrder := []string{}
	for rows.Next() {
		var groupID, groupName, optionID, optionName string
		var required bool
		var minSelections, maxSelections int
		var surcharge float64
		if err := rows.Scan(&groupID, &groupName, &required, &minSelections, &maxSelections, &optionID, &optionName, &surcharge); err != nil {
			return nil, 0, &orderPreparationError{Status: 503, Code: "modifiers_unavailable", Message: "No pudimos leer los modificadores del producto."}
		}
		current, exists := groups[groupID]
		if !exists {
			current = &group{ID: groupID, Name: groupName, Required: required, Min: minSelections, Max: maxSelections, Options: map[string]option{}}
			groups[groupID] = current
			groupOrder = append(groupOrder, groupID)
		}
		if optionID != "" {
			current.Options[optionID] = option{ID: optionID, Name: optionName, Surcharge: surcharge}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, &orderPreparationError{Status: 503, Code: "modifiers_unavailable", Message: "No pudimos completar los modificadores del producto."}
	}

	selectedByGroup := map[string][]string{}
	seen := map[string]bool{}
	for _, input := range inputs {
		groupID := strings.TrimSpace(input.GroupID)
		optionID := strings.TrimSpace(input.OptionID)
		if groupID == "" || optionID == "" {
			return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_modifier_selection", Message: "Revisa los modificadores seleccionados."}
		}
		key := groupID + ":" + optionID
		if seen[key] {
			return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_modifier_selection", Message: "Un modificador está repetido."}
		}
		seen[key] = true
		selectedByGroup[groupID] = append(selectedByGroup[groupID], optionID)
	}

	for groupID := range selectedByGroup {
		if _, exists := groups[groupID]; !exists {
			return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_modifier_selection", Message: "Un modificador no pertenece a este producto."}
		}
	}

	prepared := []preparedOrderModifier{}
	surchargeTotal := 0.0
	for _, groupID := range groupOrder {
		current := groups[groupID]
		selected := selectedByGroup[groupID]
		minimum := current.Min
		if current.Required && minimum < 1 {
			minimum = 1
		}
		if len(selected) < minimum || len(selected) > current.Max {
			return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_modifier_selection", Message: "Completa correctamente el grupo " + current.Name + "."}
		}
		for _, optionID := range selected {
			selectedOption, exists := current.Options[optionID]
			if !exists {
				return nil, 0, &orderPreparationError{Status: 400, Code: "invalid_modifier_selection", Message: "Una opción no pertenece a " + current.Name + "."}
			}
			prepared = append(prepared, preparedOrderModifier{
				GroupID: current.ID, GroupName: current.Name,
				OptionID: selectedOption.ID, Name: selectedOption.Name, Surcharge: selectedOption.Surcharge,
			})
			surchargeTotal += selectedOption.Surcharge
		}
	}
	return prepared, surchargeTotal, nil
}
