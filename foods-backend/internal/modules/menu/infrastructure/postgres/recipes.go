package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"strings"
)

func (r *Catalog) RecipeModules(ctx context.Context, s scope) (bool, bool, error) {
	var recipes, inventory bool
	err := r.db(ctx).QueryRow(ctx, `SELECT COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='recetas'),true),COALESCE((SELECT active FROM organization_modules WHERE organization_id=$1 AND module_key='inventario'),true)`, s.OrganizationID).Scan(&recipes, &inventory)
	return recipes, inventory, err
}

func (r *Catalog) LockRecipeProduct(ctx context.Context, s scope, productID string) (string, string, error) {
	var productName string
	var control string
	err := r.db(ctx).QueryRow(ctx, `SELECT name,quantity_control FROM products WHERE id=$1 AND organization_id=$2 AND active FOR UPDATE`, productID, s.OrganizationID).Scan(&productName, &control)
	return productName, control, menuError(err)
}
func (r *Catalog) RecipeIngredientActive(ctx context.Context, s scope, x domain.RecipeIngredientInput) (bool, error) {
	var ok bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_items WHERE id=$1 AND organization_id=$2 AND active)`, x.InventoryItemID, s.OrganizationID).Scan(&ok)
	return ok, menuError(err)
}
func (r *Catalog) SaveRecipe(ctx context.Context, s scope, productID string, in domain.RecipeInput) (string, error) {
	var recipeID string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO product_recipes(organization_id,product_id,yield_quantity,notes,active,created_by,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$6)
		ON CONFLICT(organization_id,product_id)
		DO UPDATE SET yield_quantity=EXCLUDED.yield_quantity,notes=EXCLUDED.notes,active=EXCLUDED.active,updated_by=EXCLUDED.updated_by,updated_at=now()
		RETURNING id
	`, s.OrganizationID, productID, in.YieldQuantity, strings.TrimSpace(in.Notes), in.Active, s.UserID).Scan(&recipeID)
	return recipeID, menuError(err)
}
func (r *Catalog) DeleteRecipeIngredients(ctx context.Context, s scope, recipeID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM product_recipe_items WHERE recipe_id=$1 AND organization_id=$2`, recipeID, s.OrganizationID)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) CreateRecipeIngredient(ctx context.Context, s scope, recipeID string, x domain.RecipeIngredientInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO product_recipe_items(organization_id,recipe_id,inventory_item_id,quantity,waste_percent) VALUES($1,$2,$3,$4,$5)`, s.OrganizationID, recipeID, x.InventoryItemID, x.Quantity, x.WastePercent)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) Recipes(ctx context.Context, s scope, f application.CatalogFilter) (application.RecipePage, error) {
	out := application.RecipePage{Items: []domain.RecipeSummary{}, Page: f.Page, PageSize: f.Size}
	q := strings.TrimSpace(f.Query)
	err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*) FROM product_recipes pr
		JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id
		WHERE pr.organization_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%')
	`, s.OrganizationID, q).Scan(&out.Total)
	if err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT pr.id,pr.product_id,p.name,pr.yield_quantity::text,pr.notes,pr.active,pr.updated_at,
		  (SELECT count(*) FROM product_recipe_items ri WHERE ri.recipe_id=pr.id AND ri.organization_id=pr.organization_id)
		FROM product_recipes pr
		JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id
		WHERE pr.organization_id=$1 AND ($2='' OR p.name ILIKE '%'||$2||'%')
		ORDER BY pr.active DESC,p.name LIMIT $3 OFFSET $4
	`, s.OrganizationID, q, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var x domain.RecipeSummary
		if err = rows.Scan(&x.ID, &x.ProductID, &x.ProductName, &x.YieldQuantity, &x.Notes, &x.Active, &x.UpdatedAt, &x.ItemCount); err != nil {
			return out, err
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}

func (r *Catalog) Recipe(ctx context.Context, s scope, productID string) (domain.RecipeView, error) {
	var out domain.RecipeView
	err := r.db(ctx).QueryRow(ctx, `
		SELECT pr.id,pr.product_id,p.name,pr.yield_quantity::text,pr.notes,pr.active,pr.updated_at
		FROM product_recipes pr JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id
		WHERE pr.product_id=$1 AND pr.organization_id=$2
	`, productID, s.OrganizationID).Scan(&out.ID, &out.ProductID, &out.ProductName, &out.YieldQuantity, &out.Notes, &out.Active, &out.UpdatedAt)
	if err != nil {
		return out, menuError(err)
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT ri.id,ri.inventory_item_id,COALESCE(p.name,ii.name),ii.unit,ri.quantity::text,ri.waste_percent::text
		FROM product_recipe_items ri
		JOIN inventory_items ii ON ii.id=ri.inventory_item_id AND ii.organization_id=ri.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE ri.recipe_id=$1 AND ri.organization_id=$2 ORDER BY COALESCE(p.name,ii.name)
	`, out.ID, s.OrganizationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Items = []domain.RecipeItemView{}
	for rows.Next() {
		var x domain.RecipeItemView
		if err = rows.Scan(&x.ID, &x.InventoryItemID, &x.Name, &x.Unit, &x.Quantity, &x.WastePercent); err != nil {
			return out, err
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
