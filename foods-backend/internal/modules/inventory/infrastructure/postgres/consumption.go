package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"math"
)

func (r *Stock) OrderBusinessDate(ctx context.Context, s scope, orderID string) (string, error) {
	var businessDate string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT (o.created_at AT TIME ZONE l.timezone)::date::text
		FROM orders o
		JOIN locations l ON l.id=o.location_id AND l.organization_id=o.organization_id
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3`,
		orderID, s.OrganizationID, s.LocationID).Scan(&businessDate)
	return businessDate, stockError(err)
}
func (r *Stock) LockQuantityProducts(ctx context.Context, s scope, productIDs []string) (map[string]domain.QuantityProduct, error) {
	// ORDER BY id keeps the lock order the per-product loop used (sorted ids).
	// FOR NO KEY UPDATE still serializes concurrent consumption of a product,
	// but does not conflict with the KEY SHARE that the order_items foreign key
	// already took in this transaction: with FOR UPDATE two simultaneous
	// orders of the same product deadlocked (40P01).
	rows, err := r.db(ctx).Query(ctx, `
			SELECT id::text,name,quantity_control
			FROM products
			WHERE id=ANY($1::uuid[]) AND organization_id=$2
			ORDER BY id
			FOR NO KEY UPDATE`, productIDs, s.OrganizationID)
	if err != nil {
		return nil, stockError(err)
	}
	defer rows.Close()
	out := map[string]domain.QuantityProduct{}
	for rows.Next() {
		var id string
		var v domain.QuantityProduct
		if err = rows.Scan(&id, &v.Name, &v.Control); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}
func (r *Stock) RecipeModules(ctx context.Context, s scope) (bool, bool, error) {
	var recipes, inventory bool
	err := r.db(ctx).QueryRow(ctx, `SELECT COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='recetas'),true),COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='inventario'),true)`, s.OrganizationID).Scan(&recipes, &inventory)
	return recipes, inventory, err
}
func (r *Stock) ActiveRecipes(ctx context.Context, s scope, productIDs []string) (map[string]domain.ActiveRecipe, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT product_id::text,id::text,yield_quantity::float8 FROM product_recipes WHERE organization_id=$1 AND product_id=ANY($2::uuid[]) AND active`, s.OrganizationID, productIDs)
	if err != nil {
		return nil, stockError(err)
	}
	defer rows.Close()
	out := map[string]domain.ActiveRecipe{}
	for rows.Next() {
		var productID string
		var v domain.ActiveRecipe
		if err = rows.Scan(&productID, &v.ID, &v.Yield); err != nil {
			return nil, err
		}
		out[productID] = v
	}
	return out, rows.Err()
}
func (r *Stock) NetRecipeUsage(ctx context.Context, s scope, orderID string) (map[string]float64, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT inventory_item_id::text,COALESCE(sum(-quantity_delta),0)::float8
		FROM stock_movements
		WHERE organization_id=$1 AND location_id=$2 AND source_type='order' AND source_id=$3
		  AND movement_type IN ('recipe_consumption','recipe_reversal')
		GROUP BY inventory_item_id
	`, s.OrganizationID, s.LocationID, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var id string
		var qty float64
		if err = rows.Scan(&id, &qty); err != nil {
			return nil, err
		}
		if math.Abs(qty) > 1e-9 {
			out[id] = qty
		}
	}
	return out, rows.Err()
}
