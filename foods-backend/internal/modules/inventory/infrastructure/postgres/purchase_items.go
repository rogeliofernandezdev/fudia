package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

func (r *Stock) Presentations(ctx context.Context, s scope, itemID string) ([]domain.InventoryPresentationOption, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id,p.presentation_type,p.units_per_presentation::text,t.name,p.is_default
		FROM inventory_presentations p JOIN inventory_presentation_types t ON t.organization_id=p.organization_id AND t.code=p.presentation_type
		WHERE p.organization_id=$1 AND p.inventory_item_id=$2 AND p.active
		ORDER BY p.is_default DESC,t.sort_order,p.units_per_presentation`, s.OrganizationID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.InventoryPresentationOption{}
	for rows.Next() {
		var p domain.InventoryPresentationOption
		if err = rows.Scan(&p.ID, &p.PresentationType, &p.UnitsPerPresentation, &p.Name, &p.IsDefault); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
