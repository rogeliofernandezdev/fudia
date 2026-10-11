package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"strings"
	"time"
)

func (r *Catalog) Availability(ctx context.Context, s scope, f application.CatalogFilter, day time.Time, excludeCombos bool) ([]domain.AvailabilitySnapshot, int, error) {
	search, categoryID, page, size := "%"+strings.TrimSpace(f.Query)+"%", f.CategoryID, f.Page, f.Size
	var total int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*)
		FROM products p
		WHERE p.organization_id=$1 AND p.active AND p.name ILIKE $2
		  AND ($3='' OR p.category_id::text=$3)
		  AND (NOT $4 OR NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id))`, s.OrganizationID, search, categoryID, excludeCombos).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id,p.name,p.price::text,p.category_id,c.name,p.image_url,p.quantity_control,
		       pa.portion_quantity,COALESCE(pa.sold_quantity,0),COALESCE(pa.manual_status,'available'),COALESCE(pa.note,''),
		       (p.available_from IS NULL OR now() >= p.available_from)
		       AND (p.available_until IS NULL OR now() <= p.available_until)
		       AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		       AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time),
		       EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id),
		       COALESCE(sb.quantity,0)::float8,ii.unit,COALESCE(ii.minimum_stock,0)::float8
		FROM products p
		JOIN locations l ON l.id=$2 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN product_availability pa
		  ON pa.organization_id=p.organization_id AND pa.location_id=$2
		 AND pa.product_id=p.id AND pa.business_date=$3
		LEFT JOIN inventory_items ii
		  ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=p.organization_id AND sb.location_id=$2 AND sb.inventory_item_id=ii.id
		WHERE p.organization_id=$1 AND p.active AND p.name ILIKE $4
		  AND ($5='' OR p.category_id::text=$5)
		  AND (NOT $6 OR NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id))
		ORDER BY c.sort_order NULLS LAST,c.name,p.name
		LIMIT $7 OFFSET $8`, s.OrganizationID, s.LocationID, day.Format("2006-01-02"), search, categoryID, excludeCombos, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []domain.AvailabilitySnapshot{}
	for rows.Next() {
		var raw domain.AvailabilitySnapshot
		item := &raw.Item
		if err = rows.Scan(&item.ProductID, &item.Name, &item.Price, &item.CategoryID, &item.CategoryName, &item.ImageURL, &item.QuantityControl, &item.PortionQuantity, &item.SoldQuantity, &raw.ManualStatus, &item.Note, &raw.ScheduleAvailable, &raw.IsCombo, &raw.InventoryQuantity, &item.InventoryUnit, &raw.MinimumStock); err != nil {
			return nil, 0, err
		}
		items = append(items, raw)
	}
	return items, total, rows.Err()
}

func (r *Catalog) ComboAvailable(ctx context.Context, s scope, comboProductID string, day time.Time) (bool, error) {
	var available bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT NOT EXISTS (
		  SELECT 1
		  FROM menu_combo_groups g
		  WHERE g.combo_product_id=$1 AND g.organization_id=$2 AND g.required
		    AND (
		      SELECT count(*)
		      FROM menu_combo_options o
		      JOIN products p ON p.id=o.option_product_id AND p.organization_id=o.organization_id
		      LEFT JOIN product_availability pa
		        ON pa.organization_id=p.organization_id AND pa.location_id=$3
		       AND pa.product_id=p.id AND pa.business_date=$4
		      LEFT JOIN inventory_items ii
		        ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		      LEFT JOIN stock_balances sb
		        ON sb.organization_id=p.organization_id AND sb.location_id=$3 AND sb.inventory_item_id=ii.id
		      JOIN locations l ON l.id=$3 AND l.organization_id=p.organization_id AND l.active
		      WHERE o.group_id=g.id
		        AND p.active
		        AND COALESCE(pa.manual_status,'available') <> 'sold_out'
		        AND (p.available_from IS NULL OR now() >= p.available_from)
		        AND (p.available_until IS NULL OR now() <= p.available_until)
		        AND (p.available_days IS NULL OR extract(dow FROM now() AT TIME ZONE l.timezone)::integer = ANY(p.available_days))
		        AND (p.available_until_time IS NULL OR (now() AT TIME ZONE l.timezone)::time <= p.available_until_time)
		        AND (
		          p.quantity_control='none'
		          OR (p.quantity_control='portions' AND pa.portion_quantity IS NOT NULL
		              AND COALESCE(pa.sold_quantity,0) < pa.portion_quantity)
		          OR (p.quantity_control='inventory' AND COALESCE(sb.quantity,0) > 0)
		        )
		    ) < GREATEST(g.min_selections, CASE WHEN g.required THEN 1 ELSE 0 END)
		)`, comboProductID, s.OrganizationID, s.LocationID, day.Format("2006-01-02")).Scan(&available)
	return available, err
}
func (r *Catalog) LockAvailabilityProduct(ctx context.Context, s scope, productID string) (string, error) {
	var quantityControl string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT quantity_control
		FROM products
		WHERE id=$1 AND organization_id=$2 AND active
		FOR UPDATE`,
		productID, s.OrganizationID).Scan(&quantityControl)
	return quantityControl, menuError(err)
}
func (r *Catalog) LockDailyAvailability(ctx context.Context, s scope, productID string, day time.Time) (*int, int, string, string, error) {
	var currentPortionQuantity *int
	var soldQuantity int
	var currentStatus string
	var currentNote string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT portion_quantity,sold_quantity,manual_status,note
		FROM product_availability
		WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date
		FOR UPDATE`,
		s.OrganizationID, s.LocationID, productID, day.Format("2006-01-02")).Scan(&currentPortionQuantity, &soldQuantity, &currentStatus, &currentNote)
	return currentPortionQuantity, soldQuantity, currentStatus, currentNote, menuError(err)
}
func (r *Catalog) UpdateAvailability(ctx context.Context, s scope, productID string, day time.Time, in domain.AvailabilityInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity,manual_status,note,updated_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (organization_id,location_id,product_id,business_date)
		DO UPDATE SET portion_quantity=EXCLUDED.portion_quantity,manual_status=EXCLUDED.manual_status,note=EXCLUDED.note,updated_by=EXCLUDED.updated_by,updated_at=now()`,
		s.OrganizationID, s.LocationID, productID, day.Format("2006-01-02"), in.PortionQuantity, in.Status, in.Note, s.UserID)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) AuditAvailability(ctx context.Context, s scope, productID string, day time.Time, in domain.AvailabilityInput, currentPortionQuantity *int, soldQuantity int, currentStatus string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,reason,created_at,metadata)
		VALUES($1,$2,$3,'product.availability_updated','product',$4,$5,clock_timestamp(),
		 jsonb_build_object('businessDate',$6::text,'previousPortionQuantity',$7::integer,
		 'portionQuantity',$8::integer,'soldQuantity',$9::integer,'previousManualStatus',$10::text,
		 'manualStatus',$11::text,'actorName',(SELECT full_name FROM users WHERE id=$3)))`,
		s.OrganizationID, s.LocationID, s.UserID, productID, in.Reason, day.Format("2006-01-02"),
		currentPortionQuantity, in.PortionQuantity, soldQuantity, currentStatus, in.Status)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) ProductExists(ctx context.Context, s scope, productID string) (bool, error) {
	var exists bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND organization_id=$2)`, productID, s.OrganizationID).Scan(&exists)
	return exists, err
}
func (r *Catalog) AvailabilityHistory(ctx context.Context, s scope, productID string, f application.CatalogFilter) (application.AvailabilityHistoryPage, error) {
	out := application.AvailabilityHistoryPage{Page: f.Page, PageSize: f.Size}
	page, size := f.Page, f.Size
	var err error
	const filter = ` FROM audit_log h WHERE h.organization_id=$1 AND h.location_id=$2 AND h.entity_id=$3 AND h.entity_type='product' AND h.action='product.availability_updated'`
	var total int
	if err = r.db(ctx).QueryRow(ctx, `SELECT count(*)`+filter, s.OrganizationID, s.LocationID, productID).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `SELECT h.id,h.created_at,h.metadata->>'businessDate',h.user_id,
	 COALESCE(h.metadata->>'actorName',(SELECT full_name FROM users WHERE id=h.user_id),'Usuario no disponible'),h.reason,
	 (h.metadata->>'previousPortionQuantity')::integer,(h.metadata->>'portionQuantity')::integer,
	 (h.metadata->>'soldQuantity')::integer,h.metadata->>'previousManualStatus',h.metadata->>'manualStatus'`+
		filter+` ORDER BY h.created_at DESC,h.id DESC LIMIT $4 OFFSET $5`, s.OrganizationID, s.LocationID, productID, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.AvailabilityHistoryItem{}
	for rows.Next() {
		var item domain.AvailabilityHistoryItem
		var createdAt time.Time
		if err = rows.Scan(&item.ID, &createdAt, &item.BusinessDate, &item.UserID, &item.UserName, &item.Reason,
			&item.PreviousPortionQuantity, &item.PortionQuantity, &item.SoldQuantity, &item.PreviousManualStatus, &item.ManualStatus); err != nil {
			return out, err
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	return out, nil
}
