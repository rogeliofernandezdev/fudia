package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"strings"
	"time"
)

var normalizedSurcharge = domain.NormalizedSurcharge

func (r *Catalog) SetComboActive(ctx context.Context, s scope, id string, in struct{ Active *bool }) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE products p SET active=$3,updated_at=now() FROM menu_combos c WHERE p.id=$2 AND p.organization_id=$1 AND c.product_id=p.id AND c.organization_id=p.organization_id`, s.OrganizationID, id, *in.Active)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) CreateComboProduct(ctx context.Context, s scope, in domain.ComboInput, sku string) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO products(organization_id,category_id,sku,name,description,price,active,quantity_control,available_from,available_until,available_days) VALUES($1,$2,$3,$4,$5,$6,true,'none',$7::timestamptz,$8::timestamptz,$9) RETURNING id`, s.OrganizationID, in.CategoryID, sku, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, in.AvailableFrom, in.AvailableUntil, in.AvailableDays).Scan(&id)
	return id, menuError(err)
}
func (r *Catalog) CreateCombo(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO menu_combos(product_id,organization_id) VALUES($1,$2)`, id, s.OrganizationID)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) UpdateCombo(ctx context.Context, s scope, id string, in domain.ComboInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE products SET category_id=$3,name=$4,description=$5,price=$6,available_from=$7::timestamptz,available_until=$8::timestamptz,available_days=$9,updated_at=now() WHERE id=$2 AND organization_id=$1`, s.OrganizationID, id, in.CategoryID, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, in.AvailableFrom, in.AvailableUntil, in.AvailableDays)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) DeleteComboGroups(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM menu_combo_groups WHERE combo_product_id=$1 AND organization_id=$2`, id, s.OrganizationID)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) CreateComboGroup(ctx context.Context, s scope, id string, group domain.ComboGroupInput, groupIndex int) (string, error) {
	var groupID string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO menu_combo_groups(organization_id,combo_product_id,name,required,min_selections,max_selections,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, s.OrganizationID, id, strings.TrimSpace(group.Name), group.Required, group.MinSelections, group.MaxSelections, groupIndex).Scan(&groupID)
	return groupID, menuError(err)
}
func (r *Catalog) CreateComboOption(ctx context.Context, s scope, id string, groupID string, option domain.ComboOptionInput, optionIndex int) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO menu_combo_options(organization_id,group_id,option_product_id,surcharge,sort_order,default_quota) SELECT $1,$2,p.id,$4,$5,$6 FROM products p WHERE p.id=$3 AND p.organization_id=$1 AND p.active AND p.id<>$7`, s.OrganizationID, groupID, option.ProductID, normalizedSurcharge(option.Surcharge), optionIndex, option.Quota, id)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) BusinessDate(ctx context.Context, s scope) (time.Time, error) {
	var day time.Time
	err := r.db(ctx).QueryRow(ctx, `SELECT (now() AT TIME ZONE timezone)::date FROM locations WHERE id=$1 AND organization_id=$2 AND active`, s.LocationID, s.OrganizationID).Scan(&day)
	return day, err
}
func (r *Catalog) Combos(ctx context.Context, s scope, f application.CatalogFilter) (application.ComboPage, error) {
	out := application.ComboPage{Items: []domain.ComboSummary{}, Page: f.Page, PageSize: f.Size}
	search, page, size := "%"+strings.TrimSpace(f.Query)+"%", f.Page, f.Size
	status := f.Status
	var total int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM menu_combos c JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id WHERE c.organization_id=$1 AND p.name ILIKE $2 AND ($3='' OR ($3='active' AND p.active) OR ($3='inactive' AND NOT p.active))`, s.OrganizationID, search, status).Scan(&total)
	if err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT p.id,p.name,p.description,p.price::text,p.active,count(DISTINCT g.id),count(o.id) FROM menu_combos c JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id LEFT JOIN menu_combo_groups g ON g.combo_product_id=c.product_id AND g.organization_id=c.organization_id LEFT JOIN menu_combo_options o ON o.group_id=g.id AND o.organization_id=c.organization_id WHERE c.organization_id=$1 AND p.name ILIKE $2 AND ($3='' OR ($3='active' AND p.active) OR ($3='inactive' AND NOT p.active)) GROUP BY p.id ORDER BY p.updated_at DESC,p.name LIMIT $4 OFFSET $5`, s.OrganizationID, search, status, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.ComboSummary
		if err = rows.Scan(&v.ID, &v.Name, &v.Description, &v.Price, &v.Active, &v.GroupCount, &v.OptionCount); err != nil {
			return out, err
		}
		out.Items = append(out.Items, v)
	}
	out.Total = total
	return out, rows.Err()
}
func (r *Catalog) OrderCombos(ctx context.Context, s scope, f application.CatalogFilter, day time.Time) (application.OrderComboPage, error) {
	out := application.OrderComboPage{Items: []domain.OrderComboSummary{}, Page: f.Page, PageSize: f.Size}
	search, page, size := "%"+strings.TrimSpace(f.Query)+"%", f.Page, f.Size
	var err error
	availabilityWhere := `
		p.active
		AND (p.available_from IS NULL OR now() >= p.available_from)
		AND (p.available_until IS NULL OR now() <= p.available_until)
		AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		AND NOT EXISTS (
			SELECT 1
			FROM menu_combo_groups g
			WHERE g.combo_product_id=p.id AND g.organization_id=p.organization_id AND g.required
			  AND (
				SELECT count(*)
				FROM menu_combo_options o
				JOIN products op ON op.id=o.option_product_id AND op.organization_id=p.organization_id AND op.active
				LEFT JOIN product_availability opa ON opa.organization_id=op.organization_id AND opa.location_id=l.id
				  AND opa.product_id=op.id AND opa.business_date=$3
				WHERE o.group_id=g.id AND o.organization_id=p.organization_id
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
				  AND (
				    o.default_quota IS NULL OR
				    COALESCE((
				      SELECT sum(oi.qty)
				      FROM order_item_combo_selections ssel
				      JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
				      JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
				      WHERE ssel.organization_id=p.organization_id
				        AND oi.product_id=p.id
				        AND ssel.group_name=g.name
				        AND ssel.option_product_id=o.option_product_id
				        AND ord.location_id=l.id
				        AND ord.status<>'cancelado'
				        AND (ord.created_at AT TIME ZONE l.timezone)::date=$3::date
				    ),0) < o.default_quota
				  )
			  ) < GREATEST(g.min_selections,1)
		)`

	var total int
	countSQL := `
		SELECT count(*)
		FROM menu_combos c
		JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=$3
		WHERE c.organization_id=$1 AND p.name ILIKE $4 AND ` + availabilityWhere
	if err = r.db(ctx).QueryRow(ctx, countSQL, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search).Scan(&total); err != nil {
		return out, err
	}

	listSQL := `
		SELECT p.id,p.name,p.description,p.price::text,p.image_url,
		       (SELECT count(*) FROM menu_combo_groups g WHERE g.combo_product_id=p.id AND g.organization_id=p.organization_id),
		       (SELECT count(*) FROM menu_combo_options o JOIN menu_combo_groups g ON g.id=o.group_id
		        WHERE g.combo_product_id=p.id AND g.organization_id=p.organization_id)
		FROM menu_combos c
		JOIN products p ON p.id=c.product_id AND p.organization_id=c.organization_id
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=$3
		WHERE c.organization_id=$1 AND p.name ILIKE $4 AND ` + availabilityWhere + `
		ORDER BY p.featured DESC,p.name
		LIMIT $5 OFFSET $6`
	rows, err := r.db(ctx).Query(ctx, listSQL, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.OrderComboSummary
		if err = rows.Scan(&v.ID, &v.Name, &v.Description, &v.Price, &v.ImageURL, &v.GroupCount, &v.OptionCount); err != nil {
			return out, err
		}
		out.Items = append(out.Items, v)
	}
	out.Total = total
	return out, rows.Err()
}
func (r *Catalog) Combo(ctx context.Context, s scope, id string) (domain.Combo, error) {
	out := domain.Combo{ID: id, Groups: []domain.ComboGroup{}}
	err := r.db(ctx).QueryRow(ctx, `SELECT p.name,p.description,p.price::text,p.active,p.available_from::text,p.available_until::text,p.available_days FROM products p JOIN menu_combos c ON c.product_id=p.id AND c.organization_id=p.organization_id WHERE p.id=$1 AND p.organization_id=$2`, id, s.OrganizationID).Scan(&out.Name, &out.Description, &out.Price, &out.Active, &out.AvailableFrom, &out.AvailableUntil, &out.AvailableDays)
	if err != nil {
		return out, menuError(err)
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT g.id,g.name,g.required,g.min_selections,g.max_selections FROM menu_combo_groups g WHERE g.combo_product_id=$1 AND g.organization_id=$2 ORDER BY g.sort_order`, id, s.OrganizationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var g domain.ComboGroup
		g.Options = []domain.ComboOption{}
		if err = rows.Scan(&g.ID, &g.Name, &g.Required, &g.MinSelections, &g.MaxSelections); err != nil {
			return out, err
		}
		out.Groups = append(out.Groups, g)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	rows.Close()
	for i := range out.Groups {
		g := &out.Groups[i]
		options, err := r.db(ctx).Query(ctx, `SELECT o.option_product_id,p.name,o.surcharge::text,o.default_quota FROM menu_combo_options o JOIN products p ON p.id=o.option_product_id WHERE o.group_id=$1 AND o.organization_id=$2 ORDER BY o.sort_order`, g.ID, s.OrganizationID)
		if err != nil {
			return out, err
		}
		for options.Next() {
			var o domain.ComboOption
			if err = options.Scan(&o.ProductID, &o.Name, &o.Surcharge, &o.Quota); err != nil {
				options.Close()
				return out, err
			}
			g.Options = append(g.Options, o)
		}
		err = options.Err()
		options.Close()
		if err != nil {
			return out, err
		}
	}
	return out, nil
}
func (r *Catalog) OrderCombo(ctx context.Context, s scope, id string, day time.Time) (domain.OrderCombo, bool, error) {
	out := domain.OrderCombo{ID: id, Groups: []domain.OrderComboGroup{}}
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
		JOIN menu_combos c ON c.product_id=p.id AND c.organization_id=p.organization_id
		JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=$4
		WHERE p.id=$1 AND p.organization_id=$2`, id, s.OrganizationID, s.LocationID, day.Format("2006-01-02")).Scan(&out.Name, &out.Description, &out.Price, &out.ImageURL, &available)
	if err != nil {
		return out, false, menuError(err)
	}
	if !available {
		return out, false, nil
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT g.id,g.name,g.required,g.min_selections,g.max_selections
		FROM menu_combo_groups g
		WHERE g.combo_product_id=$1 AND g.organization_id=$2
		ORDER BY g.sort_order,g.id`, id, s.OrganizationID)
	if err != nil {
		return out, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var g domain.OrderComboGroup
		g.Options = []domain.OrderComboOption{}
		if err = rows.Scan(&g.ID, &g.Name, &g.Required, &g.MinSelections, &g.MaxSelections); err != nil {
			return out, false, err
		}
		out.Groups = append(out.Groups, g)
	}
	if err = rows.Err(); err != nil {
		return out, false, err
	}
	rows.Close()
	for i := range out.Groups {
		g := &out.Groups[i]
		options, err := r.db(ctx).Query(ctx, `
			SELECT o.option_product_id,p.name,o.surcharge::text,
			       p.active
			       AND (p.available_from IS NULL OR now() >= p.available_from)
			       AND (p.available_until IS NULL OR now() <= p.available_until)
			       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
			       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
			       AND COALESCE(pa.manual_status,'available') <> 'sold_out'
			       
			       AND (
			         o.default_quota IS NULL OR
			         COALESCE((
			           SELECT sum(oi.qty)
			           FROM order_item_combo_selections ssel
			           JOIN order_items oi ON oi.id=ssel.order_item_id AND oi.organization_id=ssel.organization_id
			           JOIN orders ord ON ord.id=oi.order_id AND ord.organization_id=oi.organization_id
			           WHERE ssel.organization_id=o.organization_id
			             AND oi.product_id=g.combo_product_id
			             AND ssel.group_name=g.name
			             AND ssel.option_product_id=o.option_product_id
			             AND ord.location_id=l.id
			             AND ord.status<>'cancelado'
			             AND (ord.created_at AT TIME ZONE l.timezone)::date=$4::date
			         ),0) < o.default_quota
			       )
			FROM menu_combo_options o
			JOIN menu_combo_groups g ON g.id=o.group_id AND g.organization_id=o.organization_id
			JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
			JOIN locations l ON l.id=$3 AND l.organization_id=o.organization_id AND l.active
			LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
			  AND pa.product_id=p.id AND pa.business_date=$4
			WHERE o.group_id=$1 AND o.organization_id=$2
			ORDER BY o.sort_order,o.id`, g.ID, s.OrganizationID, s.LocationID, day.Format("2006-01-02"))
		if err != nil {
			return out, false, err
		}
		for options.Next() {
			var o domain.OrderComboOption
			if err = options.Scan(&o.ProductID, &o.Name, &o.Surcharge, &o.Available); err != nil {
				options.Close()
				return out, false, err
			}
			g.Options = append(g.Options, o)
		}
		err = options.Err()
		options.Close()
		if err != nil {
			return out, false, err
		}
	}
	return out, available, nil
}
