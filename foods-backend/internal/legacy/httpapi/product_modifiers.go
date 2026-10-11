package httpapi

import (
	"context"
	menudomain "github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/jackc/pgx/v5"
	"net/http"
)

type modifierOptionInput = menudomain.ModifierOptionInput
type modifierGroupInput = menudomain.ModifierGroupInput
type modifierConfigInput = menudomain.ModifierConfigInput
type modifierOptionView = menudomain.ModifierOptionView
type modifierGroupView = menudomain.ModifierGroupView
type modifierConfigView = menudomain.ModifierConfigView

var validateModifierConfig = menudomain.ValidateModifierConfig

type modifierQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadProductModifiers(
	ctx context.Context,
	q modifierQuerier,
	organizationID string,
	productID string,
) (modifierConfigView, error) {
	out := modifierConfigView{ProductID: productID, Groups: []modifierGroupView{}}
	rows, err := q.Query(ctx, `
		SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
		       o.id::text,o.name,o.surcharge::text
		FROM product_modifier_groups g
		LEFT JOIN product_modifier_options o
		  ON o.group_id=g.id AND o.organization_id=g.organization_id
		WHERE g.organization_id=$1 AND g.product_id=$2
		ORDER BY g.sort_order,g.id,o.sort_order,o.id
	`, organizationID, productID)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	index := map[string]int{}
	for rows.Next() {
		var groupID, groupName string
		var required bool
		var minSelections, maxSelections int
		var optionID, optionName, surcharge *string
		if err := rows.Scan(
			&groupID, &groupName, &required, &minSelections, &maxSelections,
			&optionID, &optionName, &surcharge,
		); err != nil {
			return out, err
		}
		position, exists := index[groupID]
		if !exists {
			position = len(out.Groups)
			index[groupID] = position
			out.Groups = append(out.Groups, modifierGroupView{
				ID: groupID, Name: groupName, Required: required,
				MinSelections: minSelections, MaxSelections: maxSelections,
				Options: []modifierOptionView{},
			})
		}
		if optionID != nil && optionName != nil && surcharge != nil {
			out.Groups[position].Options = append(out.Groups[position].Options, modifierOptionView{
				ID: *optionID, Name: *optionName, Surcharge: *surcharge,
			})
		}
	}
	return out, rows.Err()
}
func (a *API) getProductModifiers(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.GetProductModifiers(w, r)
}
func (a *API) saveProductModifiers(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.SaveProductModifiers(w, r)
}
