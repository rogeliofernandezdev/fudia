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
func (r *Stock) LockQuantityProduct(ctx context.Context, s scope, productID string) (string, string, error) {
	var name string
	var control string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT name,quantity_control
			FROM products
			WHERE id=$1 AND organization_id=$2
			FOR UPDATE`, productID, s.OrganizationID).Scan(&name, &control)
	return name, control, stockError(err)
}
func (r *Stock) LockPortions(ctx context.Context, s scope, productID string, businessDate string) (*int, int, string, error) {
	var portionQuantity *int
	var soldQuantity int
	var manualStatus string
	err := r.db(ctx).QueryRow(ctx, `
				SELECT portion_quantity,sold_quantity,manual_status
				FROM product_availability
				WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date
				FOR UPDATE`, s.OrganizationID, s.LocationID, productID, businessDate).Scan(&portionQuantity, &soldQuantity, &manualStatus)
	return portionQuantity, soldQuantity, manualStatus, stockError(err)
}
func (r *Stock) UpdateSoldPortions(ctx context.Context, s scope, productID string, businessDate string, nextSold int) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
				UPDATE product_availability
				SET sold_quantity=$5,updated_at=now()
				WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date`,
		s.OrganizationID, s.LocationID, productID, businessDate, nextSold)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) LockProductInventory(ctx context.Context, s scope, productID string) (string, error) {
	var inventoryItemID string
	err := r.db(ctx).QueryRow(ctx, `
				SELECT id
				FROM inventory_items
				WHERE organization_id=$1 AND product_id=$2 AND active
				FOR UPDATE`, s.OrganizationID, productID).Scan(&inventoryItemID)
	return inventoryItemID, stockError(err)
}
func (r *Stock) RecipeModules(ctx context.Context, s scope) (bool, bool, error) {
	var recipes, inventory bool
	err := r.db(ctx).QueryRow(ctx, `SELECT COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='recetas'),true),COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='inventario'),true)`, s.OrganizationID).Scan(&recipes, &inventory)
	return recipes, inventory, err
}
func (r *Stock) ActiveRecipe(ctx context.Context, s scope, productID string) (string, float64, error) {
	var id string
	var yield float64
	err := r.db(ctx).QueryRow(ctx, `SELECT id,yield_quantity::float8 FROM product_recipes WHERE organization_id=$1 AND product_id=$2 AND active`, s.OrganizationID, productID).Scan(&id, &yield)
	return id, yield, stockError(err)
}
func (r *Stock) RecipeIngredients(ctx context.Context, s scope, recipeID string) ([]domain.RecipeIngredient, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT inventory_item_id::text,quantity::float8,waste_percent::float8 FROM product_recipe_items WHERE recipe_id=$1 AND organization_id=$2 ORDER BY inventory_item_id`, recipeID, s.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.RecipeIngredient{}
	for rows.Next() {
		var v domain.RecipeIngredient
		if err = rows.Scan(&v.ItemID, &v.Quantity, &v.WastePercent); err != nil {
			return nil, err
		}
		out = append(out, v)
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
		if math.Abs(qty) > 0.000001 {
			out[id] = qty
		}
	}
	return out, rows.Err()
}
