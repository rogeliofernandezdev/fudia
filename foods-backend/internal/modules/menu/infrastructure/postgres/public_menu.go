package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"time"
)

func (r *Catalog) PublicMenu(ctx context.Context, s scope, q, productID string, day time.Time) ([]domain.PublicMenuSnapshot, error) {
	search := "%" + q + "%"
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id::text,p.name,p.description,p.price::text,c.name,p.image_url,p.quantity_control,
		       pa.portion_quantity,COALESCE(pa.sold_quantity,0),COALESCE(pa.manual_status,'available'),
		       (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time),
		       EXISTS(SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id),
		       EXISTS(SELECT 1 FROM product_modifier_groups pmg WHERE pmg.product_id=p.id AND pmg.organization_id=p.organization_id),
		       COALESCE(sb.quantity,0)::float8,COALESCE(ii.minimum_stock,0)::float8,p.featured
		FROM products p
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id AND c.active
		LEFT JOIN product_availability pa
		  ON pa.organization_id=p.organization_id AND pa.location_id=$2
		 AND pa.product_id=p.id AND pa.business_date=$3
		LEFT JOIN inventory_items ii
		  ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=p.organization_id AND sb.location_id=$2 AND sb.inventory_item_id=ii.id
		WHERE p.organization_id=$1 AND p.active
		  AND ($4='' OR p.name ILIKE $5 OR p.description ILIKE $5 OR COALESCE(c.name,'') ILIKE $5)
		  AND ($6='' OR p.id::text=$6)
		ORDER BY p.featured DESC,c.sort_order NULLS LAST,c.name NULLS LAST,p.name
		LIMIT 30
	`, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), q, search, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.PublicMenuSnapshot{}
	for rows.Next() {
		var item domain.PublicMenuItem
		var quantityControl, manualStatus string
		var portionQuantity *int
		var soldQuantity int
		var scheduleAvailable, isCombo bool
		var inventoryQuantity, minimumStock float64
		if err := rows.Scan(
			&item.ProductID, &item.Name, &item.Description, &item.Price, &item.CategoryName, &item.ImageURL,
			&quantityControl, &portionQuantity, &soldQuantity, &manualStatus, &scheduleAvailable, &isCombo,
			&item.HasModifiers, &inventoryQuantity, &minimumStock, &item.Recommended,
		); err != nil {
			return nil, err
		}
		item.IsCombo = isCombo
		items = append(items, domain.PublicMenuSnapshot{Item: item, Availability: domain.AvailabilitySnapshot{Item: domain.AvailabilityItem{ProductID: item.ProductID, QuantityControl: quantityControl, PortionQuantity: portionQuantity, SoldQuantity: soldQuantity}, ManualStatus: manualStatus, ScheduleAvailable: scheduleAvailable, IsCombo: isCombo, InventoryQuantity: inventoryQuantity, MinimumStock: minimumStock}})
	}
	return items, rows.Err()
}
func (r *Catalog) PublicCombo(ctx context.Context, s scope, id string, day time.Time) (domain.PublicComboDetail, bool, error) {
	var out domain.PublicComboDetail
	var available bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT p.name,p.description,p.price::text,p.image_url,
		       p.active
		       AND (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		FROM products p
		JOIN menu_combos mc ON mc.product_id=p.id AND mc.organization_id=p.organization_id
		JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa
		  ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		 AND pa.product_id=p.id AND pa.business_date=$4
		WHERE p.id::text=$1 AND p.organization_id=$2
	`, id, s.OrganizationID, s.LocationID, day.Format("2006-01-02")).
		Scan(&out.Name, &out.Description, &out.Price, &out.ImageURL, &available)
	return out, available, menuError(err)
}
func (r *Catalog) PublicComboGroups(ctx context.Context, s scope, id string, day time.Time) ([]domain.PublicComboGroup, error) {
	groupRows, err := r.db(ctx).Query(ctx, `
		SELECT id::text,name,required,min_selections,max_selections
		FROM menu_combo_groups
		WHERE combo_product_id::text=$1 AND organization_id=$2
		ORDER BY sort_order,id
	`, id, s.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer groupRows.Close()

	groups := []domain.PublicComboGroup{}
	for groupRows.Next() {
		var group domain.PublicComboGroup
		if err := groupRows.Scan(
			&group.ID, &group.Name, &group.Required, &group.MinSelections, &group.MaxSelections,
		); err != nil {
			return nil, err
		}

		rows, queryErr := r.db(ctx).Query(ctx, `
			SELECT o.option_product_id::text,p.name,o.surcharge::text,
			       p.active
			       AND (p.available_from IS NULL OR now() >= p.available_from)
			       AND (p.available_until IS NULL OR now() <= p.available_until)
			       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
			       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
			       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
			       AND (
			         p.quantity_control='none'
			         OR (p.quantity_control='portions' AND pa.portion_quantity IS NOT NULL
			             AND COALESCE(pa.sold_quantity,0) < pa.portion_quantity)
			         OR (p.quantity_control='inventory' AND COALESCE((
			           SELECT sb.quantity
			           FROM inventory_items ii
			           JOIN stock_balances sb
			             ON sb.organization_id=ii.organization_id
			            AND sb.inventory_item_id=ii.id AND sb.location_id=l.id
			           WHERE ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
			         ),0) > 0)
			       )
			       AND (
			         o.default_quota IS NULL OR
			         COALESCE((
			           SELECT sum(oi.qty)
			           FROM order_item_combo_selections sel
			           JOIN order_items oi ON oi.id=sel.order_item_id AND oi.organization_id=sel.organization_id
			           JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
			           WHERE sel.organization_id=o.organization_id
			             AND oi.product_id=$5
			             AND lower(sel.group_name)=lower($6)
			             AND sel.option_product_id=o.option_product_id
			             AND ord.location_id=l.id
			             AND ord.status<>'cancelado'
			             AND (ord.created_at AT TIME ZONE l.timezone)::date=$4::date
			         ),0) < o.default_quota
			       )
			FROM menu_combo_options o
			JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
			JOIN locations l ON l.id=$3 AND l.organization_id=o.organization_id AND l.active
			LEFT JOIN product_availability pa
			  ON pa.organization_id=p.organization_id AND pa.location_id=l.id
			 AND pa.product_id=p.id AND pa.business_date=$4
			WHERE o.group_id::text=$1 AND o.organization_id=$2
			ORDER BY o.sort_order,o.id
		`, group.ID, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), id, group.Name)
		if queryErr != nil {
			return nil, queryErr
		}
		group.Options = []domain.PublicComboOption{}
		for rows.Next() {
			var option domain.PublicComboOption
			if scanErr := rows.Scan(&option.ProductID, &option.Name, &option.Surcharge, &option.Available); scanErr != nil {
				rows.Close()
				return nil, scanErr
			}
			group.Options = append(group.Options, option)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return nil, rowsErr
		}
		rows.Close()
		groups = append(groups, group)
	}
	if err := groupRows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}
func (r *Catalog) PublicModifierProduct(ctx context.Context, s scope, productID string) (bool, bool, error) {
	var exists bool
	var isCombo bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
		         SELECT 1 FROM products
		         WHERE id=$1 AND organization_id=$2 AND active
		       ),
		       EXISTS(
		         SELECT 1 FROM menu_combos
		         WHERE product_id=$1 AND organization_id=$2
		       )
	`, productID, s.OrganizationID).Scan(&exists, &isCombo)
	return exists, isCombo, menuError(err)
}
