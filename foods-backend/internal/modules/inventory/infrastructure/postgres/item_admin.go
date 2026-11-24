package postgres

import (
	"context"
	"encoding/json"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

var _ application.ItemAdminRepository = (*Stock)(nil)

// openPurchaseStatuses are purchase orders that can still receive stock.
const openPurchaseStatuses = `('draft','pending_approval','approved','partially_received')`

// itemHistorySQL is true when any document, in any location, references the article.
const itemHistorySQL = `(
  EXISTS(SELECT 1 FROM stock_movements x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM inventory_entries x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM inventory_adjustments x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM purchase_order_items x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM purchase_return_items x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM inventory_transfer_items x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM product_recipe_items x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id)
  OR EXISTS(SELECT 1 FROM stock_balances x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id AND x.quantity<>0)
)`

func (r *Stock) ItemFacts(ctx context.Context, s scope, id string, lock bool) (domain.ItemFacts, error) {
	var f domain.ItemFacts
	query := `
		SELECT ii.id,ii.product_id,p.name,COALESCE(p.name,ii.name),ii.unit,COALESCE(u.name,ii.unit),
		       COALESCE(m.family,''),COALESCE(u.countable,false),
		       ils.inventory_item_id IS NOT NULL OR sb.inventory_item_id IS NOT NULL,
		       COALESCE(ils.active,sb.inventory_item_id IS NOT NULL),COALESCE(sb.quantity,0)::float8,
		       ` + itemHistorySQL + `,
		       (SELECT count(DISTINCT po.id) FROM purchase_order_items poi
		         JOIN purchase_orders po ON po.id=poi.purchase_order_id AND po.organization_id=poi.organization_id
		         WHERE poi.organization_id=ii.organization_id AND poi.inventory_item_id=ii.id
		           AND po.location_id=$3 AND po.status IN ` + openPurchaseStatuses + `)::int,
		       (SELECT count(DISTINCT po.id) FROM purchase_order_items poi
		         JOIN purchase_orders po ON po.id=poi.purchase_order_id AND po.organization_id=poi.organization_id
		         WHERE poi.organization_id=ii.organization_id AND poi.inventory_item_id=ii.id
		           AND po.status IN ` + openPurchaseStatuses + `)::int,
		       EXISTS(SELECT 1 FROM stock_balances x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id AND x.quantity<>trunc(x.quantity))
		       OR EXISTS(SELECT 1 FROM inventory_location_settings x WHERE x.organization_id=ii.organization_id AND x.inventory_item_id=ii.id
		         AND (x.minimum_stock<>trunc(x.minimum_stock) OR x.reorder_point<>trunc(x.reorder_point) OR x.optimal_stock<>trunc(x.optimal_stock)))
		FROM inventory_items ii
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		LEFT JOIN inventory_units u ON u.organization_id=ii.organization_id AND u.code=ii.unit
		LEFT JOIN inventory_measurement_units m ON m.code=ii.unit
		LEFT JOIN inventory_location_settings ils ON ils.organization_id=ii.organization_id AND ils.location_id=$3 AND ils.inventory_item_id=ii.id
		LEFT JOIN stock_balances sb ON sb.organization_id=ii.organization_id AND sb.location_id=$3 AND sb.inventory_item_id=ii.id
		WHERE ii.organization_id=$1 AND ii.id=$2 AND ii.active
		  AND EXISTS(SELECT 1 FROM locations l WHERE l.organization_id=$1 AND l.id=$3 AND l.active)`
	if lock {
		query += ` FOR UPDATE OF ii`
	}
	err := r.db(ctx).QueryRow(ctx, query, s.OrganizationID, id, s.LocationID).Scan(
		&f.InventoryItemID, &f.ProductID, &f.ProductName, &f.Name, &f.Unit, &f.UnitName, &f.UnitFamily, &f.UnitCountable,
		&f.AssignedHere, &f.ActiveHere, &f.QuantityHere, &f.HasHistory, &f.OpenOrdersHere, &f.OpenOrdersAny, &f.FractionalAnyStock)
	if err != nil {
		return f, stockError(err)
	}
	if f.Recipes, err = r.itemRecipes(ctx, s, id); err != nil {
		return f, err
	}
	f.OtherLocations, err = r.itemOtherLocations(ctx, s, id)
	return f, err
}

func (r *Stock) itemRecipes(ctx context.Context, s scope, id string) ([]domain.ItemRecipeUse, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.name,COALESCE(ri.consumption_unit,'')
		FROM product_recipe_items ri
		JOIN product_recipes pr ON pr.id=ri.recipe_id AND pr.organization_id=ri.organization_id AND pr.active
		JOIN products p ON p.id=pr.product_id AND p.organization_id=pr.organization_id AND p.active
		WHERE ri.organization_id=$1 AND ri.inventory_item_id=$2
		ORDER BY p.name`, s.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ItemRecipeUse{}
	for rows.Next() {
		var item domain.ItemRecipeUse
		if err := rows.Scan(&item.ProductName, &item.ConsumptionUnit); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Stock) itemOtherLocations(ctx context.Context, s scope, id string) ([]domain.ItemLocationUse, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT l.id,l.name,ils.active,COALESCE(sb.quantity,0)::float8
		FROM inventory_location_settings ils
		JOIN locations l ON l.id=ils.location_id AND l.organization_id=ils.organization_id AND l.active
		LEFT JOIN stock_balances sb ON sb.organization_id=ils.organization_id AND sb.location_id=ils.location_id AND sb.inventory_item_id=ils.inventory_item_id
		WHERE ils.organization_id=$1 AND ils.inventory_item_id=$2 AND ils.location_id<>$3
		ORDER BY l.name`, s.OrganizationID, id, s.LocationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ItemLocationUse{}
	for rows.Next() {
		var item domain.ItemLocationUse
		var quantity float64
		if err := rows.Scan(&item.LocationID, &item.Name, &item.Active, &quantity); err != nil {
			return nil, err
		}
		item.Quantity = domain.FormatQuantity(quantity)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Stock) ItemLevels(ctx context.Context, s scope, id string) (string, string, error) {
	var minimum, optimal float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT COALESCE(ils.minimum_stock,ii.minimum_stock)::float8,
		       COALESCE(ils.optimal_stock,COALESCE(ils.reorder_point,COALESCE(ils.minimum_stock,ii.minimum_stock)))::float8
		FROM inventory_items ii
		LEFT JOIN inventory_location_settings ils ON ils.organization_id=ii.organization_id AND ils.location_id=$3 AND ils.inventory_item_id=ii.id
		WHERE ii.organization_id=$1 AND ii.id=$2`, s.OrganizationID, id, s.LocationID).Scan(&minimum, &optimal)
	return domain.FormatQuantity(minimum), domain.FormatQuantity(optimal), stockError(err)
}

func (r *Stock) ItemPresentations(ctx context.Context, s scope, id string) ([]domain.ItemPresentation, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT ip.id,ip.presentation_type,ip.units_per_presentation::float8,pt.name,ip.is_default,
		       ip.presentation_type='unit' AND ip.units_per_presentation=1,
		       EXISTS(SELECT 1 FROM purchase_order_items x WHERE x.organization_id=ip.organization_id AND x.presentation_id=ip.id)
		       OR EXISTS(SELECT 1 FROM purchase_receipt_items x WHERE x.organization_id=ip.organization_id AND x.presentation_id=ip.id)
		       OR EXISTS(SELECT 1 FROM inventory_entries x WHERE x.organization_id=ip.organization_id AND x.presentation_id=ip.id)
		FROM inventory_presentations ip
		JOIN inventory_presentation_types pt ON pt.organization_id=ip.organization_id AND pt.code=ip.presentation_type
		WHERE ip.organization_id=$1 AND ip.inventory_item_id=$2 AND ip.active
		ORDER BY ip.presentation_type='unit' AND ip.units_per_presentation=1 DESC,ip.is_default DESC,pt.sort_order,ip.units_per_presentation`, s.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ItemPresentation{}
	for rows.Next() {
		var item domain.ItemPresentation
		var units float64
		if err := rows.Scan(&item.ID, &item.PresentationType, &units, &item.Name, &item.IsDefault, &item.IsBase, &item.Used); err != nil {
			return nil, err
		}
		item.UnitsPerPresentation = domain.FormatQuantity(units)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Stock) UnitTarget(ctx context.Context, s scope, code string) (domain.UnitChangeTarget, error) {
	out := domain.UnitChangeTarget{Code: code}
	err := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM inventory_units WHERE organization_id=$1 AND code=$2),
		       COALESCE((SELECT name FROM inventory_units WHERE organization_id=$1 AND code=$2),$2),
		       COALESCE((SELECT family FROM inventory_measurement_units WHERE code=$2),''),
		       COALESCE((SELECT countable FROM inventory_units WHERE organization_id=$1 AND code=$2),false)`,
		s.OrganizationID, code).Scan(&out.Exists, &out.Name, &out.Family, &out.Countable)
	return out, err
}

func (r *Stock) IncompatiblePresentations(ctx context.Context, s scope, id, unit string) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT pt.name||' x '||trim(trailing '.' FROM trim(trailing '0' FROM ip.units_per_presentation::text))
		FROM inventory_presentations ip
		JOIN inventory_presentation_types pt ON pt.organization_id=ip.organization_id AND pt.code=ip.presentation_type
		LEFT JOIN inventory_units u ON u.organization_id=ip.organization_id AND u.code=$3
		WHERE ip.organization_id=$1 AND ip.inventory_item_id=$2 AND ip.active
		  AND NOT (ip.presentation_type='unit' AND ip.units_per_presentation=1)
		  AND (
		    NOT EXISTS(SELECT 1 FROM inventory_unit_combinations c
		      WHERE c.organization_id=ip.organization_id AND c.unit=$3 AND c.presentation_type=ip.presentation_type)
		    OR (COALESCE(u.countable,false) AND ip.units_per_presentation<>trunc(ip.units_per_presentation))
		  )
		ORDER BY pt.sort_order,ip.units_per_presentation`, s.OrganizationID, id, unit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// LocationsWithoutPermission lists the other active locations that keep the
// article and where the user lacks the permission: a company-wide change made
// from one location must not reach locations the user cannot manage.
func (r *Stock) LocationsWithoutPermission(ctx context.Context, s scope, id, permission string) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT l.name
		FROM inventory_location_settings ils
		JOIN locations l ON l.id=ils.location_id AND l.organization_id=ils.organization_id AND l.active
		WHERE ils.organization_id=$1 AND ils.inventory_item_id=$2 AND ils.location_id<>$3
		  AND NOT EXISTS(
		    SELECT 1 FROM user_roles ur
		    JOIN roles ro ON ro.id=ur.role_id AND ro.organization_id=$1 AND ro.active
		    JOIN users u ON u.id=ur.user_id AND u.organization_id=ro.organization_id AND u.active
		    WHERE ur.user_id=$4 AND (ur.location_id=l.id OR ro.system_key='administrator')
		      AND $5 = ANY(ro.permissions))
		ORDER BY l.name`, s.OrganizationID, id, s.LocationID, s.UserID, permission)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// ChangeUnit relabels the article: quantities keep their numbers. Recipes that
// consume it in its own unit and manual entries move to the new code with it.
func (r *Stock) ChangeUnit(ctx context.Context, s scope, id, from, to string) error {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE inventory_items SET unit=$3,updated_at=now() WHERE organization_id=$1 AND id=$2 AND active`, s.OrganizationID, id, to)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return application.ErrNotFound
	}
	if _, err = r.db(ctx).Exec(ctx, `UPDATE product_recipe_items SET consumption_unit=$4 WHERE organization_id=$1 AND inventory_item_id=$2 AND consumption_unit=$3`, s.OrganizationID, id, from, to); err != nil {
		return err
	}
	_, err = r.db(ctx).Exec(ctx, `UPDATE inventory_entries SET unit=$4 WHERE organization_id=$1 AND inventory_item_id=$2 AND unit=$3`, s.OrganizationID, id, from, to)
	return err
}

func (r *Stock) SetLocationActive(ctx context.Context, s scope, id string, active bool) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO inventory_location_settings(organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock,active)
		SELECT ii.organization_id,$3,ii.id,ii.minimum_stock,ii.minimum_stock,ii.minimum_stock,$4
		FROM inventory_items ii WHERE ii.organization_id=$1 AND ii.id=$2
		ON CONFLICT(location_id,inventory_item_id) DO UPDATE SET active=EXCLUDED.active,updated_at=now()`,
		s.OrganizationID, id, s.LocationID, active)
	return err
}

// HidePresentation stops offering a presentation; rows are never deleted so
// purchases and entries keep their reference. The base one is never hidden and
// becomes the default when the hidden one was.
func (r *Stock) HidePresentation(ctx context.Context, s scope, id, presentationID string) (bool, error) {
	var base, wasDefault bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT ip.presentation_type='unit' AND ip.units_per_presentation=1,ip.is_default
		FROM inventory_presentations ip
		WHERE ip.organization_id=$1 AND ip.inventory_item_id=$2 AND ip.id=$3 AND ip.active
		FOR UPDATE`, s.OrganizationID, id, presentationID).Scan(&base, &wasDefault)
	if err != nil || base {
		return base, stockError(err)
	}
	if _, err = r.db(ctx).Exec(ctx, `UPDATE inventory_presentations SET active=false,is_default=false,updated_at=now() WHERE organization_id=$1 AND id=$2`, s.OrganizationID, presentationID); err != nil || !wasDefault {
		return false, err
	}
	_, err = r.db(ctx).Exec(ctx, `UPDATE inventory_presentations SET is_default=true,updated_at=now()
		WHERE organization_id=$1 AND inventory_item_id=$2 AND active AND presentation_type='unit' AND units_per_presentation=1`, s.OrganizationID, id)
	return false, err
}

func (r *Stock) AssignableItems(ctx context.Context, s scope, query string, limit int) ([]domain.AssignableItem, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT ii.id,ii.product_id,COALESCE(p.name,ii.name),ii.unit,COALESCE(u.name,ii.unit),
		       here.inventory_item_id IS NOT NULL,
		       COALESCE((SELECT jsonb_agg(l.name ORDER BY l.name)
		         FROM inventory_location_settings o
		         JOIN locations l ON l.id=o.location_id AND l.organization_id=o.organization_id AND l.active
		         WHERE o.organization_id=ii.organization_id AND o.inventory_item_id=ii.id AND o.active AND o.location_id<>$2),'[]'::jsonb)
		FROM inventory_items ii
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		LEFT JOIN inventory_units u ON u.organization_id=ii.organization_id AND u.code=ii.unit
		LEFT JOIN inventory_location_settings here ON here.organization_id=ii.organization_id AND here.location_id=$2 AND here.inventory_item_id=ii.id
		WHERE ii.organization_id=$1 AND ii.active
		  AND (p.id IS NULL OR (p.active AND p.quantity_control='inventory'))
		  AND COALESCE(here.active,false)=false
		  AND (p.name ILIKE $3 OR ii.name ILIKE $3)
		ORDER BY COALESCE(p.name,ii.name),ii.id
		LIMIT $4`, s.OrganizationID, s.LocationID, containsPattern(query), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AssignableItem{}
	for rows.Next() {
		var item domain.AssignableItem
		var locations []byte
		if err := rows.Scan(&item.InventoryItemID, &item.ProductID, &item.Name, &item.Unit, &item.UnitName, &item.InactiveHere, &locations); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(locations, &item.Locations); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Stock) AuditItem(ctx context.Context, s scope, action, id string, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata)
		VALUES($1,$2,NULLIF($3,'')::uuid,$4,'inventory_item',$5,$6::jsonb)`, s.OrganizationID, s.LocationID, s.UserID, action, id, string(payload))
	return err
}
