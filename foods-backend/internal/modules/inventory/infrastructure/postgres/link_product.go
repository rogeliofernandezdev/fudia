package postgres

import (
	"context"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

// linkableProductFilter selects active products without physical inventory that are
// neither combos nor products with an active recipe.
const linkableProductFilter = `
	p.organization_id=$1 AND p.active AND p.quantity_control IN ('none','portions')
	AND NOT EXISTS(SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id)
	AND NOT EXISTS(SELECT 1 FROM product_recipes pr WHERE pr.product_id=p.id AND pr.organization_id=p.organization_id AND pr.active)`

func (r *Stock) LinkableProducts(ctx context.Context, s scope, query string, limit int) ([]domain.LinkableProduct, error) {
	search := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(query) + "%"
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id,p.sku,p.name,c.name,p.quantity_control
		FROM products p
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		WHERE `+linkableProductFilter+`
		  AND (p.name ILIKE $2 OR p.sku ILIKE $2)
		ORDER BY p.name
		LIMIT $3`, s.OrganizationID, search, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.LinkableProduct{}
	for rows.Next() {
		var item domain.LinkableProduct
		if err = rows.Scan(&item.ProductID, &item.SKU, &item.Name, &item.CategoryName, &item.QuantityControl); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Stock) LockLinkableProduct(ctx context.Context, s scope, productID string) (domain.LinkProductCandidate, error) {
	var out domain.LinkProductCandidate
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.id,p.sku,p.name,p.quantity_control,p.category_id,
		  EXISTS(SELECT 1 FROM product_recipes pr WHERE pr.product_id=p.id AND pr.organization_id=p.organization_id AND pr.active),
		  EXISTS(SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id)
		FROM products p
		WHERE p.id=$1 AND p.organization_id=$2 AND p.active
		FOR UPDATE OF p`, productID, s.OrganizationID,
	).Scan(&out.ID, &out.SKU, &out.Name, &out.QuantityControl, &out.CategoryID, &out.HasRecipe, &out.IsCombo)
	return out, stockError(err)
}

// ProductInOpenOrders mirrors Catalog's open-order guard: changing the quantity
// source while an order can still be cancelled would reverse it into the wrong ledger.
func (r *Stock) ProductInOpenOrders(ctx context.Context, s scope, productID string) (bool, error) {
	var used bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM orders o
			JOIN order_items oi ON oi.order_id=o.id AND oi.organization_id=o.organization_id
			WHERE o.organization_id=$1
			  AND o.status IN ('nuevo','confirmado','preparando','listo')
			  AND (
			    oi.product_id=$2
			    OR EXISTS(
			      SELECT 1 FROM order_item_combo_selections cs
			      WHERE cs.organization_id=oi.organization_id AND cs.order_item_id=oi.id AND cs.option_product_id=$2
			    )
			  )
		)`, s.OrganizationID, productID).Scan(&used)
	return used, stockError(err)
}

// MarkProductInventoryControlled keeps the commercial record (name, price, image,
// history) and only changes where its quantity comes from. Physical merchandise is
// retail; a Carta category receiving it widens to «both», as in CreateRetailProduct.
func (r *Stock) MarkProductInventoryControlled(ctx context.Context, s scope, product domain.LinkProductCandidate) error {
	if _, err := r.db(ctx).Exec(ctx, `
		UPDATE products SET quantity_control='inventory',product_type='retail',updated_at=now()
		WHERE id=$1 AND organization_id=$2`, product.ID, s.OrganizationID); err != nil {
		return err
	}
	if product.CategoryID == nil {
		return nil
	}
	_, err := r.db(ctx).Exec(ctx, `UPDATE menu_categories SET product_scope='both'
		WHERE organization_id=$1 AND id=$2 AND product_scope='prepared'`, s.OrganizationID, *product.CategoryID)
	return err
}
