package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

func (r *Stock) RecipeUnitRule(ctx context.Context, s scope, itemID, unit string) (domain.RecipeUnitRule, error) {
	var out domain.RecipeUnitRule
	err := r.db(ctx).QueryRow(ctx, `SELECT i.unit,COALESCE(c.code,''),COALESCE(bm.family,''),COALESCE(cm.family,''),
 COALESCE(bm.reference_factor,1)::text,COALESCE(cm.reference_factor,1)::text
 FROM inventory_items i JOIN inventory_units b ON b.organization_id=i.organization_id AND b.code=i.unit
 LEFT JOIN inventory_measurement_units bm ON bm.code=b.code
 LEFT JOIN inventory_units c ON c.organization_id=i.organization_id AND c.code=COALESCE(NULLIF($3,''),i.unit)
 LEFT JOIN inventory_measurement_units cm ON cm.code=c.code
 WHERE i.id=$1 AND i.organization_id=$2 AND i.active FOR SHARE OF i,b`, itemID, s.OrganizationID, unit).
		Scan(&out.InventoryUnit, &out.ConsumptionUnit, &out.InventoryFamily, &out.ConsumptionFamily, &out.InventoryFactor, &out.ConsumptionFactor)
	return out, stockError(err)
}
