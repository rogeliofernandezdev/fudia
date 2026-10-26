package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type scope = requestscope.Scope
type Stock struct{ pool *pgxpool.Pool }

func NewStock(pool *pgxpool.Pool) *Stock                  { return &Stock{pool} }
func (r *Stock) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }

// containsPattern builds an ILIKE pattern that matches the text literally:
// % and _ typed by the user are not wildcards. Backslash is PostgreSQL's
// default LIKE escape character.
func containsPattern(query string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.TrimSpace(query))
	return "%" + escaped + "%"
}
func stockError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.StockRepository = (*Stock)(nil)

func (r *Stock) UnitExists(ctx context.Context, s scope, code string) (bool, error) {
	var valid bool
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_units WHERE organization_id=$1 AND code=$2)`, s.OrganizationID, code).Scan(&valid)
	return valid, err
}
func (r *Stock) PresentationRule(ctx context.Context, s scope, item, typeCode string) (domain.PresentationRule, error) {
	var rule domain.PresentationRule
	err := r.db(ctx).QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM inventory_items i JOIN inventory_presentation_types t ON t.organization_id=i.organization_id AND t.code=$3
 WHERE i.organization_id=$1 AND i.id=$2 AND (t.code='unit' OR EXISTS(SELECT 1 FROM inventory_unit_combinations c WHERE c.organization_id=i.organization_id AND c.unit=i.unit AND c.presentation_type=t.code))),
 COALESCE(u.countable,false),COALESCE(u.name,'')
 FROM (SELECT 1) one
 LEFT JOIN inventory_items i ON i.organization_id=$1 AND i.id=$2
 LEFT JOIN inventory_units u ON u.organization_id=i.organization_id AND u.code=i.unit`, s.OrganizationID, item, typeCode).Scan(&rule.Allowed, &rule.Countable, &rule.UnitName)
	return rule, err
}
func (r *Stock) EnsurePresentation(ctx context.Context, s scope, inventoryItemID, presentationType string, unitsPerPresentation float64) (string, error) {
	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO inventory_presentations(
			organization_id,inventory_item_id,presentation_type,units_per_presentation,active
		)
		VALUES($1,$2,'unit',1,true)
		ON CONFLICT (organization_id,inventory_item_id,presentation_type,units_per_presentation)
		DO UPDATE SET active=true,updated_at=now()`,
		s.OrganizationID, inventoryItemID); err != nil {
		return "", err
	}
	var presentationID string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO inventory_presentations(
			organization_id,inventory_item_id,presentation_type,units_per_presentation,active
		)
		VALUES($1,$2,$3,$4,true)
		ON CONFLICT (organization_id,inventory_item_id,presentation_type,units_per_presentation)
		DO UPDATE SET active=true,updated_at=now()
		RETURNING id`,
		s.OrganizationID, inventoryItemID, presentationType, unitsPerPresentation).
		Scan(&presentationID)
	return presentationID, err
}
func (r *Stock) DefaultPresentation(ctx context.Context, s scope, item, id string) error {
	if _, err := r.db(ctx).Exec(ctx, `SELECT id FROM inventory_items WHERE organization_id=$1 AND id=$2 FOR UPDATE`, s.OrganizationID, item); err != nil {
		return err
	}
	if _, err := r.db(ctx).Exec(ctx, `UPDATE inventory_presentations SET is_default=false WHERE organization_id=$1 AND inventory_item_id=$2 AND is_default`, s.OrganizationID, item); err != nil {
		return err
	}
	tag, err := r.db(ctx).Exec(ctx, `UPDATE inventory_presentations SET is_default=true,updated_at=now() WHERE organization_id=$1 AND inventory_item_id=$2 AND id=$3 AND active`, s.OrganizationID, item, id)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("presentation not found")
	}
	return err
}
func (r *Stock) LockBalance(ctx context.Context, s scope, inventoryItemID string) (domain.InventoryBalanceLock, error) {
	var out domain.InventoryBalanceLock
	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
		SELECT $1,$2,ii.id,0,0
		FROM inventory_items ii
		WHERE ii.id=$3 AND ii.organization_id=$1 AND ii.active
		ON CONFLICT(location_id,inventory_item_id) DO NOTHING
	`, s.OrganizationID, s.LocationID, inventoryItemID); err != nil {
		return out, stockError(err)
	}
	err := r.db(ctx).QueryRow(ctx, `
		SELECT ii.product_id,COALESCE(p.name,ii.name),ii.unit,sb.quantity::float8,sb.average_unit_cost::float8
		FROM inventory_items ii
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		JOIN stock_balances sb ON sb.organization_id=ii.organization_id AND sb.inventory_item_id=ii.id
		  AND sb.location_id=$2
		WHERE ii.id=$3 AND ii.organization_id=$1 AND ii.active
		FOR UPDATE OF sb,ii
	`, s.OrganizationID, s.LocationID, inventoryItemID).Scan(&out.ProductID, &out.Name, &out.Unit, &out.Quantity, &out.AverageUnitCost)
	return out, stockError(err)
}
func (r *Stock) SaveMovement(ctx context.Context, s scope, m application.Movement) error {
	var userID *string
	if strings.TrimSpace(s.UserID) != "" {
		v := s.UserID
		userID = &v
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO stock_movements(
		  organization_id,location_id,product_id,inventory_item_id,movement_type,
		  quantity_delta,balance_after,source_type,source_id,note,created_by,
		  unit_cost,value_delta,balance_value_after
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, s.OrganizationID, s.LocationID, m.ProductID, m.ItemID, m.Type, m.Delta, m.Balance, m.SourceType, m.SourceID, m.Note, userID, m.UnitCost, m.ValueDelta, m.BalanceValue)
	return err
}
func (r *Stock) RetailCategoryAllowed(ctx context.Context, s scope, organizationID string, in domain.InventoryEntryInput) (bool, error) {
	var categoryAllowed bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM menu_categories
				WHERE id=$1 AND organization_id=$2 AND active
			)`,
		in.NewProduct.CategoryID, organizationID,
	).Scan(&categoryAllowed)
	return categoryAllowed, stockError(err)
}
func (r *Stock) CreateRetailProduct(ctx context.Context, s scope, organizationID string, in domain.InventoryEntryInput) (string, string, string, error) {
	var productID string
	var result domain.InventoryCatalogCreateResult
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO products(organization_id,category_id,sku,name,description,price,active,product_type,quantity_control,service_destination)
			VALUES($1,$2,COALESCE(NULLIF($3,''),'PROD-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,8))),$4,$5,$6,true,'retail','inventory','direct')
			RETURNING id,sku,name`,
		organizationID, in.NewProduct.CategoryID, in.NewProduct.SKU,
		in.NewProduct.Name, in.NewProduct.Description, in.NewProduct.Price,
	).Scan(&productID, &result.SKU, &result.Name)
	if err == nil {
		// Una categoría de Carta que recibe mercadería pasa a «both»: el uso se amplía, no se bloquea.
		_, err = r.db(ctx).Exec(ctx, `UPDATE menu_categories SET product_scope='both'
			WHERE organization_id=$1 AND id=$2 AND product_scope='prepared'`,
			organizationID, in.NewProduct.CategoryID)
	}
	return productID, result.SKU, result.Name, err
}
func (r *Stock) LinkRetailProduct(ctx context.Context, s scope, organizationID string, productID string, in domain.InventoryEntryInput, result domain.InventoryCatalogCreateResult) (string, string, error) {
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO inventory_items(
				organization_id,sku,name,unit,minimum_stock,active,product_id
			)
			VALUES($1,'SELL-'||upper(substr(replace($2::text,'-',''),1,12)),$3,$4,$5,true,$2::uuid)
			RETURNING id,unit`,
		organizationID, productID, result.Name, in.Unit, in.MinimumStockValue(),
	).Scan(&result.InventoryItemID, &result.Unit)
	return result.InventoryItemID, result.Unit, err
}
func (r *Stock) CreateIngredient(ctx context.Context, s scope, organizationID string, in domain.InventoryEntryInput, result domain.InventoryCatalogCreateResult) (string, string, string, error) {
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO inventory_items(
				organization_id,sku,name,unit,minimum_stock,active
			)
			VALUES(
				$1,
				'ING-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,10)),
				$2,$3,$4,true
			)
			RETURNING id,sku,unit`,
		organizationID, result.Name, in.Unit, in.MinimumStockValue(),
	).Scan(&result.InventoryItemID, &result.SKU, &result.Unit)
	return result.InventoryItemID, result.SKU, result.Unit, err
}
func (r *Stock) LockEntryItem(ctx context.Context, s scope, in domain.InventoryEntryInput) (string, string, string, string, *string, *string, error) {
	var inventoryItemID string
	var itemSKU string
	var itemName string
	var inventoryUnit string
	var linkedProductID *string
	var quantityControl *string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT ii.id,ii.sku,COALESCE(p.name,ii.name),ii.unit,p.id,p.quantity_control
			FROM inventory_items ii
			LEFT JOIN products p
			  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
			WHERE ii.id=$1
			  AND ii.organization_id=$2
			  AND ii.active
			  AND (p.id IS NULL OR p.active)
			FOR UPDATE OF ii`,
		in.InventoryItemID, s.OrganizationID,
	).Scan(&inventoryItemID, &itemSKU, &itemName, &inventoryUnit, &linkedProductID, &quantityControl)
	return inventoryItemID, itemSKU, itemName, inventoryUnit, linkedProductID, quantityControl, stockError(err)
}
func (r *Stock) LockEntryProduct(ctx context.Context, s scope, in domain.InventoryEntryInput) (string, string, string, string, error) {
	var existingProductID string
	var itemSKU string
	var itemName string
	var quantityControl string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT id,sku,name,quantity_control
			FROM products
			WHERE id=$1 AND organization_id=$2 AND active
			FOR UPDATE`, in.ProductID, s.OrganizationID).
		Scan(&existingProductID, &itemSKU, &itemName, &quantityControl)
	return existingProductID, itemSKU, itemName, quantityControl, stockError(err)
}
func (r *Stock) LinkEntryProduct(ctx context.Context, s scope, in domain.InventoryEntryInput, existingProductID string, itemName string) (string, string, error) {
	var inventoryItemID string
	var inventoryUnit string
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO inventory_items(
				organization_id,sku,name,unit,minimum_stock,active,product_id
			)
			VALUES($1,'SELL-'||upper(substr(replace($2::text,'-',''),1,12)),$3,$4,COALESCE($5::numeric,0),true,$2::uuid)
			ON CONFLICT (organization_id,product_id) WHERE product_id IS NOT NULL
			DO UPDATE SET
				name=EXCLUDED.name,
				minimum_stock=COALESCE($5::numeric,inventory_items.minimum_stock),
				active=true,
				updated_at=now()
			RETURNING id,unit`,
		s.OrganizationID, existingProductID, itemName, in.Unit, in.MinimumStock,
	).Scan(&inventoryItemID, &inventoryUnit)
	return inventoryItemID, inventoryUnit, stockError(err)
}
func (r *Stock) EntryLocationSettings(ctx context.Context, s scope, inventoryItemID string, in domain.InventoryEntryInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO inventory_location_settings(organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock)
		VALUES($1,$2,$3,$4,$4,$4)
		ON CONFLICT(location_id,inventory_item_id) DO UPDATE SET minimum_stock=EXCLUDED.minimum_stock,
		  reorder_point=GREATEST(inventory_location_settings.reorder_point,EXCLUDED.minimum_stock),
		  optimal_stock=GREATEST(inventory_location_settings.optimal_stock,EXCLUDED.minimum_stock),updated_at=now()
	`, s.OrganizationID, s.LocationID, inventoryItemID, in.MinimumStockValue())
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) EnsureEntryBalance(ctx context.Context, s scope, inventoryItemID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
		VALUES($1,$2,$3,0)
		ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
		s.OrganizationID, s.LocationID, inventoryItemID)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) LockEntryBalance(ctx context.Context, s scope, inventoryItemID string) (float64, float64, error) {
	var current float64
	var averageCost float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT quantity::float8,average_unit_cost::float8
		FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
		FOR UPDATE`, s.OrganizationID, s.LocationID, inventoryItemID).Scan(&current, &averageCost)
	return current, averageCost, stockError(err)
}
func (r *Stock) CreateEntry(ctx context.Context, s scope, productID *string, inventoryItemID string, in domain.InventoryEntryInput, presentationID string) (string, error) {
	var entryID string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO inventory_entries(
			organization_id,location_id,product_id,inventory_item_id,
			quantity,unit,presentation_id,presentation_type,units_per_presentation,note,created_by
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		s.OrganizationID, s.LocationID, productID, inventoryItemID,
		in.Quantity, in.Unit, presentationID, in.PresentationType,
		in.UnitsPerPresentation, in.Note, s.UserID).Scan(&entryID)
	return entryID, stockError(err)
}
func (r *Stock) LockAdjustmentItem(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (*string, string, string, error) {
	var productID *string
	var itemName string
	var unit string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT ii.product_id,COALESCE(p.name,ii.name),ii.unit
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE ii.id=$1 AND ii.organization_id=$2 AND ii.active
		  AND (p.id IS NULL OR p.active)
		FOR UPDATE OF ii`, in.InventoryItemID, s.OrganizationID).
		Scan(&productID, &itemName, &unit)
	return productID, itemName, unit, stockError(err)
}
func (r *Stock) EnsureAdjustmentBalance(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
		VALUES($1,$2,$3,0)
		ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
		s.OrganizationID, s.LocationID, in.InventoryItemID)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) LockAdjustmentBalance(ctx context.Context, s scope, in domain.InventoryAdjustmentInput) (float64, float64, error) {
	var stockBefore float64
	var averageCost float64
	err := r.db(ctx).QueryRow(ctx, `
		SELECT quantity::float8,average_unit_cost::float8
		FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
		FOR UPDATE`, s.OrganizationID, s.LocationID, in.InventoryItemID).Scan(&stockBefore, &averageCost)
	return stockBefore, averageCost, stockError(err)
}
func (r *Stock) UpdateAdjustmentBalance(ctx context.Context, s scope, in domain.InventoryAdjustmentInput, stockAfter float64, newAverage float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE stock_balances
		SET quantity=$4,average_unit_cost=$5,updated_at=now()
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, in.InventoryItemID, stockAfter, newAverage)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) CreateAdjustment(ctx context.Context, s scope, in domain.InventoryAdjustmentInput, stockBefore float64, stockAfter float64) (string, time.Time, error) {
	var adjustmentID string
	var createdAt time.Time
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO inventory_adjustments(
			organization_id,location_id,inventory_item_id,movement_type,reason,
			quantity,stock_before,stock_after,observation,created_by
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id,created_at`,
		s.OrganizationID, s.LocationID, in.InventoryItemID, in.MovementType, in.Reason,
		in.Quantity, stockBefore, stockAfter, in.Observation, s.UserID).
		Scan(&adjustmentID, &createdAt)
	return adjustmentID, createdAt, stockError(err)
}
func (r *Stock) Items(ctx context.Context, s scope, f application.StockFilter) (application.ItemsPage, error) {
	out := application.ItemsPage{Page: f.Page, PageSize: f.Size}
	page, size := f.Page, f.Size
	search := containsPattern(f.Query)
	var total int
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*)
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE ii.organization_id=$1 AND ii.active
		  AND (p.name ILIKE $2 OR ii.name ILIKE $2 OR p.sku ILIKE $2 OR ii.sku ILIKE $2)`,
		s.OrganizationID, search).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT ii.id,p.id,COALESCE(p.sku,ii.sku),COALESCE(p.name,ii.name),
		       CASE WHEN p.id IS NULL THEN 'ingredient' ELSE 'product' END,
		       c.name,COALESCE(p.active,ii.active),ii.unit,
		       COALESCE(sb.quantity,0)::text,
		       COALESCE(ils.minimum_stock,ii.minimum_stock)::text,
		       COALESCE(ils.reorder_point,COALESCE(ils.minimum_stock,ii.minimum_stock))::text,
		       COALESCE(ils.optimal_stock,COALESCE(ils.reorder_point,COALESCE(ils.minimum_stock,ii.minimum_stock)))::text,
		       COALESCE(sb.average_unit_cost,0)::text,
		       round(COALESCE(sb.quantity,0)*COALESCE(sb.average_unit_cost,0),4)::text,
		       CASE
		         WHEN COALESCE(sb.quantity,0)<=0 THEN 'out'
		         WHEN COALESCE(ils.minimum_stock,ii.minimum_stock)>0 AND COALESCE(sb.quantity,0)<=COALESCE(ils.minimum_stock,ii.minimum_stock) THEN 'low'
		         ELSE 'ok'
		       END,
		       COALESCE(sb.updated_at,ii.updated_at)
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		LEFT JOIN menu_categories c
		  ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=ii.organization_id
		 AND sb.location_id=$2
		 AND sb.inventory_item_id=ii.id
		LEFT JOIN inventory_location_settings ils
		  ON ils.organization_id=ii.organization_id
		 AND ils.location_id=$2
		 AND ils.inventory_item_id=ii.id
		WHERE ii.organization_id=$1 AND ii.active
		  AND (p.name ILIKE $3 OR ii.name ILIKE $3 OR p.sku ILIKE $3 OR ii.sku ILIKE $3)
		ORDER BY COALESCE(p.active,ii.active) DESC,COALESCE(p.name,ii.name),ii.id
		LIMIT $4 OFFSET $5`,
		s.OrganizationID, s.LocationID, search, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.InventoryItemView{}
	for rows.Next() {
		var item domain.InventoryItemView
		if err := rows.Scan(
			&item.InventoryItemID, &item.ProductID, &item.SKU, &item.Name, &item.Kind,
			&item.CategoryName, &item.Active, &item.Unit, &item.Quantity, &item.MinimumStock,
			&item.ReorderPoint, &item.OptimalStock, &item.AverageUnitCost, &item.StockValue,
			&item.Status, &item.UpdatedAt,
		); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Stock) Products(ctx context.Context, s scope, f application.StockFilter) (application.ProductsPage, error) {
	out := application.ProductsPage{Page: f.Page, PageSize: f.Size}
	page, size := f.Page, f.Size
	search := containsPattern(f.Query)
	rows, err := r.db(ctx).Query(ctx, `
		SELECT ii.id,p.id,COALESCE(p.sku,ii.sku),COALESCE(p.name,ii.name),
		       CASE WHEN p.id IS NULL THEN 'ingredient' ELSE 'product' END,
		       c.name,p.quantity_control,ii.unit,COALESCE(sb.quantity,0)::text,
		       COALESCE(ils.minimum_stock,ii.minimum_stock)::text,
		       COALESCE(ils.reorder_point,COALESCE(ils.minimum_stock,ii.minimum_stock))::text,
		       COALESCE(ils.optimal_stock,COALESCE(ils.reorder_point,COALESCE(ils.minimum_stock,ii.minimum_stock)))::text,
		       COALESCE(sb.average_unit_cost,0)::text,
		       COALESCE((
		         SELECT jsonb_agg(
		           jsonb_build_object(
		             'id',ip.id,
		             'presentationType',ip.presentation_type,
			             'unitsPerPresentation',ip.units_per_presentation::text,
			             'name',pt.name,
			             'isDefault',ip.is_default
		           )
		           ORDER BY
		             ip.is_default DESC,pt.sort_order,
		             ip.units_per_presentation
		         )
		         FROM inventory_presentations ip
			         JOIN inventory_presentation_types pt ON pt.organization_id=ip.organization_id AND pt.code=ip.presentation_type
		         WHERE ip.organization_id=ii.organization_id
		           AND ip.inventory_item_id=ii.id
		           AND ip.active
		       ),'[]'::jsonb),
		       count(*) OVER()
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		LEFT JOIN menu_categories c
		  ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=ii.organization_id
		 AND sb.location_id=$2
		 AND sb.inventory_item_id=ii.id
		LEFT JOIN inventory_location_settings ils
		  ON ils.organization_id=ii.organization_id
		 AND ils.location_id=$2
		 AND ils.inventory_item_id=ii.id
		WHERE ii.organization_id=$1
		  AND ii.active
		  AND (p.id IS NULL OR (
		    p.active
		    AND p.quantity_control='inventory'
		    AND NOT EXISTS (
		      SELECT 1
		      FROM menu_combos mc
		      WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id
		    )
		  ))
		  AND (p.name ILIKE $3 OR ii.name ILIKE $3 OR p.sku ILIKE $3 OR ii.sku ILIKE $3)
		ORDER BY COALESCE(p.name,ii.name),ii.id
		LIMIT $4 OFFSET $5`, s.OrganizationID, s.LocationID, search, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.InventoryProductOption{}
	total := 0
	for rows.Next() {
		var item domain.InventoryProductOption
		var presentationsJSON []byte
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.SKU, &item.Name, &item.Kind, &item.CategoryName,
			&item.QuantityControl, &item.Unit, &item.Quantity, &item.MinimumStock,
			&item.ReorderPoint, &item.OptimalStock, &item.AverageUnitCost, &presentationsJSON, &total,
		); err != nil {
			return out, err
		}
		if err := json.Unmarshal(presentationsJSON, &item.Presentations); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Stock) Movements(ctx context.Context, s scope, f application.StockFilter) (application.MovementsPage, error) {
	out := application.MovementsPage{Page: f.Page, PageSize: f.Size}
	page, size := f.Page, f.Size
	inventoryItemID, productID, movementType, sourceType, from, to := f.InventoryItemID, f.ProductID, f.MovementType, f.SourceType, f.From, f.To
	where := `sm.organization_id=$1 AND sm.location_id=$2
	  AND ($3='' OR sm.inventory_item_id::text=$3)
	  AND ($4='' OR sm.product_id::text=$4)
	  AND ($5='' OR sm.movement_type=$5)
	  AND ($6='' OR sm.source_type=$6)
	  AND ($7='' OR sm.created_at >= ($7::date::timestamp AT TIME ZONE (SELECT timezone FROM locations WHERE id=$2 AND organization_id=$1)))
	  AND ($8='' OR sm.created_at < (($8::date + interval '1 day')::timestamp AT TIME ZONE (SELECT timezone FROM locations WHERE id=$2 AND organization_id=$1)))`
	var total int
	if err := r.db(ctx).QueryRow(ctx, "SELECT count(*) FROM stock_movements sm WHERE "+where, s.OrganizationID, s.LocationID, inventoryItemID, productID, movementType, sourceType, from, to).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT sm.id,sm.inventory_item_id,sm.product_id,COALESCE(p.name,ii.name),
		       sm.movement_type,ia.movement_type,ia.reason,sm.quantity_delta::text,
		       COALESCE(ia.stock_before,sm.balance_after-sm.quantity_delta)::text,sm.balance_after::text,
		       sm.unit_cost::text,sm.value_delta::text,sm.balance_value_after::text,
		       sm.source_type,sm.source_id,
		       -- Readable references: document codes generated by the database are internal.
		       COALESCE(
		         CASE
		           WHEN sm.source_type='order' THEN (
		             SELECT 'Pedido · '||CASE o.channel
		               WHEN 'salon' THEN COALESCE('Mesa '||tb.name,'Salón')
		               WHEN 'mostrador' THEN 'Recojo en local'
		               WHEN 'recojo' THEN 'Recojo en local'
		               WHEN 'delivery' THEN 'Delivery'
		               WHEN 'whatsapp' THEN 'WhatsApp'
		               ELSE 'Venta' END
		             FROM orders o
		             LEFT JOIN tables tb ON tb.id=o.table_id AND tb.organization_id=o.organization_id
		             WHERE o.id=sm.source_id AND o.organization_id=sm.organization_id)
		           WHEN sm.source_type='inventory_entry' THEN 'Entrada manual'
		           WHEN sm.source_type='inventory_adjustment' THEN 'Ajuste de inventario'
		           WHEN sm.source_type='purchase_receipt' THEN (
		             SELECT 'Recepción de compra · '||sp.name FROM purchase_receipts pr
		             JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		             JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		             WHERE pr.id=sm.source_id AND pr.organization_id=sm.organization_id)
		           WHEN sm.source_type='purchase_return' THEN (
		             SELECT CASE pr.kind WHEN 'receipt_correction' THEN 'Corrección de compra · ' ELSE 'Devolución de compra · ' END||sp.name
		             FROM purchase_returns pr
		             JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		             JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		             WHERE pr.id=sm.source_id AND pr.organization_id=sm.organization_id)
		           WHEN sm.source_type='inventory_transfer' THEN (
		             SELECT 'Transferencia · '||fl.name||' → '||tl.name FROM inventory_transfers t
		             JOIN locations fl ON fl.id=t.from_location_id AND fl.organization_id=t.organization_id
		             JOIN locations tl ON tl.id=t.to_location_id AND tl.organization_id=t.organization_id
		             WHERE t.id=sm.source_id AND t.organization_id=sm.organization_id)
		         END,'Referencia no disponible'),
		       sm.note,COALESCE(u.full_name,'Usuario no disponible'),sm.created_at
		FROM stock_movements sm
		JOIN inventory_items ii ON ii.id=sm.inventory_item_id AND ii.organization_id=sm.organization_id
		LEFT JOIN products p ON p.id=sm.product_id AND p.organization_id=sm.organization_id
		LEFT JOIN inventory_adjustments ia ON ia.id=sm.source_id AND ia.organization_id=sm.organization_id AND sm.source_type='inventory_adjustment'
		LEFT JOIN users u ON u.id=sm.created_by
		WHERE `+where+`
		ORDER BY sm.created_at DESC,sm.sequence_number DESC LIMIT $9 OFFSET $10`,
		s.OrganizationID, s.LocationID, inventoryItemID, productID, movementType, sourceType, from, to, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.InventoryMovementView{}
	for rows.Next() {
		var item domain.InventoryMovementView
		if err := rows.Scan(
			&item.ID, &item.InventoryItemID, &item.ProductID, &item.ItemName, &item.MovementType, &item.AdjustmentType, &item.Reason,
			&item.QuantityDelta, &item.BalanceBefore, &item.BalanceAfter, &item.UnitCost, &item.ValueDelta, &item.BalanceValueAfter,
			&item.SourceType, &item.SourceID, &item.SourceReference, &item.Note, &item.CreatedByName, &item.CreatedAt,
		); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}

func (r *Stock) SaveBalance(ctx context.Context, s scope, itemID string, quantity, averageCost float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, itemID, quantity, averageCost)
	return tag.RowsAffected(), err
}
