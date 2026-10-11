package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"strings"
)

func (r *Orders) ExistingSelections(ctx context.Context, s scope, itemID string) ([]domain.PreparedOrderSelection, error) {
	rows, err := r.db(ctx).Query(ctx, `
					SELECT group_id::text,group_name,option_product_id::text,option_name,surcharge::float8
					FROM order_item_combo_selections
					WHERE order_item_id=$1 AND organization_id=$2
					ORDER BY created_at,id`, itemID, s.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PreparedOrderSelection{}
	for rows.Next() {
		var v domain.PreparedOrderSelection
		if err = rows.Scan(&v.GroupID, &v.GroupName, &v.ProductID, &v.Name, &v.Surcharge); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Orders) ComboGroups(ctx context.Context, s scope, productID string) (map[string]*domain.ComboGroup, []string, error) {
	groups := map[string]*domain.ComboGroup{}
	groupOrder := []string{}
	rows, err := r.db(ctx).Query(ctx, `
			SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
			       COALESCE(o.option_product_id::text,''),COALESCE(op.name,''),COALESCE(o.surcharge::float8,0),
			       CASE WHEN op.id IS NULL THEN false ELSE
			         op.active
			         AND (op.available_from IS NULL OR now() >= op.available_from)
			         AND (op.available_until IS NULL OR now() <= op.available_until)
			         AND (op.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(op.available_days))
			         AND (op.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= op.available_until_time)
			         AND COALESCE(opa.manual_status,'available') <> 'sold_out'
			         AND (
			           op.quantity_control='none'
			           OR (op.quantity_control='portions' AND opa.portion_quantity IS NOT NULL
			               AND COALESCE(opa.sold_quantity,0) < opa.portion_quantity)
			           OR (op.quantity_control='inventory' AND COALESCE((
			               SELECT sb.quantity
			               FROM inventory_items ii
			               JOIN stock_balances sb ON sb.organization_id=ii.organization_id
			                 AND sb.inventory_item_id=ii.id AND sb.location_id=l.id
			               WHERE ii.organization_id=op.organization_id AND ii.product_id=op.id AND ii.active
			             ),0) > 0)
			         )
			       END
			FROM menu_combo_groups g
			JOIN locations l ON l.id=$3 AND l.organization_id=$2 AND l.active
			LEFT JOIN menu_combo_options o ON o.group_id=g.id AND o.organization_id=g.organization_id
			LEFT JOIN products op ON op.id=o.option_product_id AND op.organization_id=g.organization_id
			LEFT JOIN product_availability opa ON opa.organization_id=g.organization_id AND opa.location_id=l.id
			  AND opa.product_id=op.id AND opa.business_date=(now() AT TIME ZONE l.timezone)::date
			WHERE g.combo_product_id=$1 AND g.organization_id=$2
			ORDER BY g.sort_order,g.id,o.sort_order,o.id`, productID, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var groupID, groupName, optionID, optionName string
		var required bool
		var minSel, maxSel int
		var surcharge float64
		var available bool
		if err := rows.Scan(&groupID, &groupName, &required, &minSel, &maxSel, &optionID, &optionName, &surcharge, &available); err != nil {
			rows.Close()
			return nil, nil, err
		}
		group, ok := groups[groupID]
		if !ok {
			group = &domain.ComboGroup{ID: groupID, Name: groupName, Required: required, Min: minSel, Max: maxSel, Options: map[string]domain.ComboOption{}}
			groups[groupID] = group
			groupOrder = append(groupOrder, groupID)
		}
		if optionID != "" {
			group.Options[optionID] = domain.ComboOption{ProductID: optionID, Name: optionName, Surcharge: surcharge, Available: available}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return groups, groupOrder, nil
}
func (r *Orders) ModifierGroups(ctx context.Context, s scope, productID string) (map[string]*domain.ModifierGroup, []string, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT g.id::text,g.name,g.required,g.min_selections,g.max_selections,
		       COALESCE(o.id::text,''),COALESCE(o.name,''),COALESCE(o.surcharge::float8,0)
		FROM product_modifier_groups g
		LEFT JOIN product_modifier_options o
		  ON o.group_id=g.id AND o.organization_id=g.organization_id
		WHERE g.organization_id=$1 AND g.product_id=$2
		ORDER BY g.sort_order,g.id,o.sort_order,o.id
	`, s.OrganizationID, productID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	groups := map[string]*domain.ModifierGroup{}
	groupOrder := []string{}
	for rows.Next() {
		var groupID, groupName, optionID, optionName string
		var required bool
		var minSelections, maxSelections int
		var surcharge float64
		if err := rows.Scan(&groupID, &groupName, &required, &minSelections, &maxSelections, &optionID, &optionName, &surcharge); err != nil {
			return nil, nil, err
		}
		current, exists := groups[groupID]
		if !exists {
			current = &domain.ModifierGroup{ID: groupID, Name: groupName, Required: required, Min: minSelections, Max: maxSelections, Options: map[string]domain.ModifierOption{}}
			groups[groupID] = current
			groupOrder = append(groupOrder, groupID)
		}
		if optionID != "" {
			current.Options[optionID] = domain.ModifierOption{ID: optionID, Name: optionName, Surcharge: surcharge}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return groups, groupOrder, nil
}
func (r *Orders) ProductIsCombo(ctx context.Context, s scope, productID *string) (bool, error) {
	var isCombo bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM menu_combos WHERE product_id=$1 AND organization_id=$2)`, *productID, s.OrganizationID).Scan(&isCombo)
	return isCombo, orderError(err)
}
func (r *Orders) ProductForOrder(ctx context.Context, s scope, productID *string) (string, float64, bool, error) {
	var productName string
	var productPrice float64
	var available bool
	err := r.db(ctx).QueryRow(ctx, `
				SELECT p.name,p.price::float8,
				       p.active
				       AND (p.available_from IS NULL OR now() >= p.available_from)
				       AND (p.available_until IS NULL OR now() <= p.available_until)
				       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
				       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
				       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
				FROM products p
				JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
				LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
				  AND pa.product_id=p.id AND pa.business_date=(now() AT TIME ZONE l.timezone)::date
				WHERE p.id=$1 AND p.organization_id=$2`, *productID, s.OrganizationID, s.LocationID).Scan(&productName, &productPrice, &available)
	return productName, productPrice, available, orderError(err)
}
func (r *Orders) ExistingComboItem(ctx context.Context, s scope, in domain.OrderItemInput, existingOrderID string) (string, float64, string, string, error) {
	var existingName string
	var existingUnitPrice float64
	var existingProductID string
	var existingType string
	err := r.db(ctx).QueryRow(ctx, `
				SELECT name,unit_price::float8,COALESCE(product_id::text,''),item_type
				FROM order_items
				WHERE id=$1 AND order_id=$2 AND organization_id=$3`,
		strings.TrimSpace(in.ID), existingOrderID, s.OrganizationID).Scan(&existingName, &existingUnitPrice, &existingProductID, &existingType)
	return existingName, existingUnitPrice, existingProductID, existingType, orderError(err)
}
func (r *Orders) ComboForOrder(ctx context.Context, s scope, productID *string) (string, float64, bool, error) {
	var comboName string
	var basePrice float64
	var comboAvailable bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT p.name,p.price::float8,
			       p.active
			       AND (p.available_from IS NULL OR now() >= p.available_from)
			       AND (p.available_until IS NULL OR now() <= p.available_until)
			       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
			       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
			       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
			FROM products p
			JOIN menu_combos mc ON mc.product_id=p.id AND mc.organization_id=p.organization_id
			JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
			LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
			  AND pa.product_id=p.id AND pa.business_date=(now() AT TIME ZONE l.timezone)::date
			WHERE p.id=$1 AND p.organization_id=$2`, *productID, s.OrganizationID, s.LocationID).Scan(&comboName, &basePrice, &comboAvailable)
	return comboName, basePrice, comboAvailable, orderError(err)
}
func (r *Orders) ExistingOptionUsage(ctx context.Context, s scope, existingOrderID string, key domain.ComboOptionUsageKey) (float64, error) {
	var existingUsage float64
	err := r.db(ctx).QueryRow(ctx, `
				SELECT COALESCE(sum(oi.qty),0)::float8
				FROM order_item_combo_selections ssel
				JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
				WHERE ssel.organization_id=$1
				  AND oi.order_id=$2
				  AND oi.product_id=$3
				  AND lower(ssel.group_name)=lower($4)
				  AND ssel.option_product_id=$5`,
		s.OrganizationID, existingOrderID, key.ComboProductID, key.GroupName, key.ProductID).Scan(&existingUsage)
	return existingUsage, orderError(err)
}
func (r *Orders) LockOptionQuota(ctx context.Context, s scope, key domain.ComboOptionUsageKey) (*int, string, error) {
	var quota *int
	var optionName string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT o.default_quota,p.name
			FROM menu_combo_options o
			JOIN menu_combo_groups g ON g.id=o.group_id AND g.organization_id=o.organization_id
			JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
			WHERE g.combo_product_id=$1
			  AND lower(g.name)=lower($2)
			  AND o.option_product_id=$3
			  AND o.organization_id=$4
			FOR UPDATE OF o`,
		key.ComboProductID, key.GroupName, key.ProductID, s.OrganizationID).Scan(&quota, &optionName)
	return quota, optionName, orderError(err)
}
func (r *Orders) ExternalOptionUsage(ctx context.Context, s scope, key domain.ComboOptionUsageKey, existingOrderID string) (float64, error) {
	var externalUsage float64
	err := r.db(ctx).QueryRow(ctx, `
			SELECT COALESCE(sum(oi.qty),0)::float8
			FROM order_item_combo_selections ssel
			JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
			JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
			JOIN locations l ON l.id=ord.location_id AND l.organization_id=ord.organization_id
			WHERE ssel.organization_id=$1
			  AND oi.product_id=$2
			  AND lower(ssel.group_name)=lower($3)
			  AND ssel.option_product_id=$4
			  AND ord.location_id=$5
			  AND ord.status<>'cancelado'
			  AND (ord.created_at AT TIME ZONE l.timezone)::date=(now() AT TIME ZONE l.timezone)::date
			  AND ($6='' OR ord.id::text<>$6)`,
		s.OrganizationID, key.ComboProductID, key.GroupName, key.ProductID, s.LocationID, existingOrderID).Scan(&externalUsage)
	return externalUsage, orderError(err)
}
