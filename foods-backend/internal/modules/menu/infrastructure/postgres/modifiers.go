package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"strings"
)

func (r *Catalog) DeleteModifiers(ctx context.Context, s scope, productID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		DELETE FROM product_modifier_groups
		WHERE organization_id=$1 AND product_id=$2
	`, s.OrganizationID, productID)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) CreateModifierGroup(ctx context.Context, s scope, productID string, group domain.ModifierGroupInput, groupIndex int) (string, error) {
	var groupID string
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO product_modifier_groups(
			  organization_id,product_id,name,required,min_selections,max_selections,sort_order
			)
			VALUES($1,$2,$3,$4,$5,$6,$7)
			RETURNING id
		`, s.OrganizationID, productID, strings.TrimSpace(group.Name),
		group.Required, group.MinSelections, group.MaxSelections, groupIndex).Scan(&groupID)
	return groupID, menuError(err)
}
func (r *Catalog) CreateModifierOption(ctx context.Context, s scope, groupID string, option domain.ModifierOptionInput, surcharge string, optionIndex int) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
				INSERT INTO product_modifier_options(
				  organization_id,group_id,name,surcharge,sort_order
				)
				VALUES($1,$2,$3,$4,$5)
			`, s.OrganizationID, groupID, strings.TrimSpace(option.Name),
		surcharge, optionIndex)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) Modifiers(ctx context.Context, s scope, productID string) (domain.ModifierConfigView, error) {
	out := domain.ModifierConfigView{ProductID: productID, Groups: []domain.ModifierGroupView{}}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
		       o.id::text,o.name,o.surcharge::text
		FROM product_modifier_groups g
		LEFT JOIN product_modifier_options o
		  ON o.group_id=g.id AND o.organization_id=g.organization_id
		WHERE g.organization_id=$1 AND g.product_id=$2
		ORDER BY g.sort_order,g.id,o.sort_order,o.id
	`, s.OrganizationID, productID)
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
			out.Groups = append(out.Groups, domain.ModifierGroupView{
				ID: groupID, Name: groupName, Required: required,
				MinSelections: minSelections, MaxSelections: maxSelections,
				Options: []domain.ModifierOptionView{},
			})
		}
		if optionID != nil && optionName != nil && surcharge != nil {
			out.Groups[position].Options = append(out.Groups[position].Options, domain.ModifierOptionView{
				ID: *optionID, Name: *optionName, Surcharge: *surcharge,
			})
		}
	}
	return out, rows.Err()
}
func (r *Catalog) SupportsModifiers(ctx context.Context, s scope, productID string) (bool, error) {
	var exists, combo bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		         SELECT 1 FROM products
		         WHERE id=$1 AND organization_id=$2
		       ),
		       EXISTS(
		         SELECT 1 FROM menu_combos
		         WHERE product_id=$1 AND organization_id=$2
		       )
	`, productID, s.OrganizationID).Scan(&exists, &combo)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, application.ErrNotFound
	}
	return !combo, nil
}
