package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/menu/application"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type scope = requestscope.Scope
type Catalog struct{ pool *pgxpool.Pool }

func NewCatalog(pool *pgxpool.Pool) *Catalog                { return &Catalog{pool} }
func (r *Catalog) db(ctx context.Context) database.Executor { return database.ExecutorFor(ctx, r.pool) }
func menuError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.CatalogRepository = (*Catalog)(nil)

func (r *Catalog) CreateCategory(ctx context.Context, s scope, in domain.CategoryInput, productScope string) (domain.Category, error) {
	var c domain.Category
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO menu_categories(organization_id,name,sort_order,product_scope)
		VALUES($1,$2,$3,$4)
		RETURNING id,name,sort_order,active,product_scope`,
		s.OrganizationID, strings.TrimSpace(in.Name), in.SortOrder, productScope,
	).Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active, &c.ProductScope)
	return c, menuError(err)
}
func (r *Catalog) IncompatibleCategoryProducts(ctx context.Context, s scope, id string, incompatibleType string) (int, error) {
	var incompatibleProducts int
	err := r.db(ctx).QueryRow(ctx, `
			SELECT count(*)
			FROM products
			WHERE organization_id=$1 AND category_id=$2 AND product_type=$3`,
		s.OrganizationID, id, incompatibleType,
	).Scan(&incompatibleProducts)
	return incompatibleProducts, menuError(err)
}
func (r *Catalog) UpdateCategory(ctx context.Context, s scope, id string, in domain.CategoryInput, active bool, productScope string) (domain.Category, error) {
	var c domain.Category
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE menu_categories
		SET name=$3,sort_order=$4,active=$5,
		    product_scope=COALESCE(NULLIF($6,''),product_scope)
		WHERE id=$2 AND organization_id=$1
		RETURNING id,name,sort_order,active,product_scope`,
		s.OrganizationID, id, strings.TrimSpace(in.Name), in.SortOrder, active, productScope,
	).Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active, &c.ProductScope)
	return c, menuError(err)
}
func (r *Catalog) ActiveCategoryProducts(ctx context.Context, s scope, id string) (int, error) {
	var products int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM products WHERE organization_id=$1 AND category_id=$2 AND active`, s.OrganizationID, id).Scan(&products)
	return products, menuError(err)
}
func (r *Catalog) DeactivateCategory(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE menu_categories SET active=false WHERE organization_id=$1 AND id=$2 AND active`, s.OrganizationID, id)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) ActivateCategory(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE menu_categories SET active=true WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) Product(ctx context.Context, s scope, id string) (domain.Product, error) {
	var p domain.Product
	err := r.db(ctx).QueryRow(ctx, `SELECT p.id,p.sku,p.name,p.description,p.category_id,c.name,p.price::text,p.active,p.product_type,p.quantity_control,p.image_url,p.prep_minutes,p.allergens,p.featured,p.cost_price::text,p.available_from::text,p.available_until::text,p.available_days,p.available_until_time::text,p.service_destination FROM products p LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id WHERE p.organization_id=$1 AND p.id=$2`, s.OrganizationID, id).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime, &p.ServiceDestination)
	return p, menuError(err)
}
func (r *Catalog) CreateProduct(ctx context.Context, s scope, in domain.ProductInput, createFeatured bool) (domain.Product, error) {
	var p domain.Product
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO products(organization_id,category_id,sku,name,description,price,active,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price,available_from,available_until,available_days,available_until_time,service_destination) VALUES($1,$2,$3,$4,$5,$6,true,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) RETURNING id,sku,name,description,category_id,price::text,active,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price::text,available_from::text,available_until::text,available_days,available_until_time::text,service_destination`, s.OrganizationID, in.CategoryID, strings.TrimSpace(in.SKU), strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, in.ProductType, in.QuantityControl, in.ImageURL, in.PrepMinutes, in.Allergens, createFeatured, in.CostPrice, in.AvailableFrom, in.AvailableUntil, in.AvailableDays, in.AvailableUntilTime, in.ServiceDestination).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime, &p.ServiceDestination)
	return p, menuError(err)
}
func (r *Catalog) InitialPortions(ctx context.Context, s scope, in domain.ProductInput, p domain.Product) (string, error) {
	var createdID string
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO product_availability(organization_id,location_id,product_id,business_date,portion_quantity,manual_status,updated_by)
			SELECT $1,l.id,$3,(now() AT TIME ZONE l.timezone)::date,$4,'available',$5
			FROM locations l WHERE l.organization_id=$1 AND l.id=$2 AND l.active
			RETURNING product_id`, s.OrganizationID, s.LocationID, p.ID, *in.InitialPortionQuantity, s.UserID).Scan(&createdID)
	return createdID, menuError(err)
}
func (r *Catalog) ProductCreatedAudit(ctx context.Context, s scope, p domain.Product, in domain.ProductInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata)
		VALUES($1,$2,$3,'product.created','product',$4,jsonb_build_object('initialPortionQuantity',$5::integer))`, s.OrganizationID, s.LocationID, s.UserID, p.ID, in.InitialPortionQuantity)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) LockProduct(ctx context.Context, s scope, id string) (string, string, string, string, error) {
	var currentSKU string
	var currentControl string
	var currentProductType string
	var currentDestination string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT sku,quantity_control,product_type,service_destination
		FROM products
		WHERE organization_id=$1 AND id=$2
		FOR UPDATE`, s.OrganizationID, id).Scan(&currentSKU, &currentControl, &currentProductType, &currentDestination)
	return currentSKU, currentControl, currentProductType, currentDestination, menuError(err)
}
func (r *Catalog) HasStockMovements(ctx context.Context, s scope, id string) (bool, error) {
	var used bool
	err := r.db(ctx).QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM stock_movements
					WHERE organization_id=$1 AND product_id=$2
				)`, s.OrganizationID, id).Scan(&used)
	return used, menuError(err)
}
func (r *Catalog) HasPortionSales(ctx context.Context, s scope, id string) (bool, error) {
	var used bool
	err := r.db(ctx).QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM product_availability
					WHERE organization_id=$1 AND product_id=$2 AND sold_quantity>0
				)`, s.OrganizationID, id).Scan(&used)
	return used, menuError(err)
}
func (r *Catalog) UpdateProduct(ctx context.Context, s scope, id string, in domain.ProductInput, skuValue string, active bool, featured bool) (domain.Product, error) {
	var p domain.Product
	err := r.db(ctx).QueryRow(ctx, `UPDATE products SET category_id=$3,sku=$4,name=$5,description=$6,price=$7,active=$8,product_type=$9,quantity_control=$10,image_url=$11,prep_minutes=$12,allergens=$13,featured=$14,cost_price=$15,available_from=$16,available_until=$17,available_days=$18,available_until_time=$19,service_destination=$20,updated_at=now() WHERE organization_id=$1 AND id=$2 RETURNING id,sku,name,description,category_id,price::text,active,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price::text,available_from::text,available_until::text,available_days,available_until_time::text,service_destination`, s.OrganizationID, id, in.CategoryID, skuValue, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, active, in.ProductType, in.QuantityControl, in.ImageURL, in.PrepMinutes, in.Allergens, featured, in.CostPrice, in.AvailableFrom, in.AvailableUntil, in.AvailableDays, in.AvailableUntilTime, in.ServiceDestination).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime, &p.ServiceDestination)
	return p, menuError(err)
}
func (r *Catalog) SetProductActive(ctx context.Context, s scope, id string, in struct{ Active *bool }) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE products SET active=$3,updated_at=now()
		WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id, *in.Active)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) DeactivateProduct(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE products SET active=false,updated_at=now() WHERE organization_id=$1 AND id=$2 AND active`, s.OrganizationID, id)
	return tag.RowsAffected(), menuError(err)
}
func (r *Catalog) Categories(ctx context.Context, s scope, f application.CatalogFilter) (application.CategoryPage, error) {
	out := application.CategoryPage{Page: f.Page, PageSize: f.Size}
	includeInactive, productType, page, size := f.IncludeInactive, f.ProductType, f.Page, f.Size
	var total int
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*)
		FROM menu_categories
		WHERE organization_id=$1
		  AND ($2 OR active)
		  AND ($3='' OR product_scope='both' OR product_scope=$3)`,
		s.OrganizationID, includeInactive, productType,
	).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT c.id,c.name,c.sort_order,c.active,c.product_scope,count(p.id)
		FROM menu_categories c
		LEFT JOIN products p ON p.category_id=c.id AND p.organization_id=c.organization_id
		WHERE c.organization_id=$1
		  AND ($2 OR c.active)
		  AND ($3='' OR c.product_scope='both' OR c.product_scope=$3)
		GROUP BY c.id
		ORDER BY c.sort_order,c.name
		LIMIT $4 OFFSET $5`,
		s.OrganizationID, includeInactive, productType, size, (page-1)*size,
	)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.Category{}
	for rows.Next() {
		var c domain.Category
		if err = rows.Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active, &c.ProductScope, &c.ProductCount); err != nil {
			return out, err
		}
		items = append(items, c)
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Catalog) Products(ctx context.Context, s scope, f application.CatalogFilter) (application.ProductPage, error) {
	out := application.ProductPage{Page: f.Page, PageSize: f.Size}
	search, status, categoryID, page, size := "%"+strings.TrimSpace(f.Query)+"%", f.Status, f.CategoryID, f.Page, f.Size
	var total int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM products p WHERE p.organization_id=$1 AND (p.name ILIKE $2 OR p.sku ILIKE $2) AND ($3='' OR p.category_id::text=$3) AND ($4='' OR ($4='active' AND p.active) OR ($4='inactive' AND NOT p.active)) AND NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id)`, s.OrganizationID, search, categoryID, status).Scan(&total)
	if err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.id,p.sku,p.name,p.description,p.category_id,c.name,p.price::text,p.active,p.product_type,p.quantity_control,p.image_url,p.prep_minutes,p.allergens,p.featured,p.cost_price::text,p.available_from::text,p.available_until::text,p.available_days,p.available_until_time::text,p.service_destination,
		       CASE
		         WHEN p.quantity_control='none' THEN NULL
		         WHEN NOT p.active OR pa.manual_status='sold_out'
		           OR (p.available_from IS NOT NULL AND now()<p.available_from)
		           OR (p.available_until IS NOT NULL AND now()>p.available_until)
		           OR (p.available_days IS NOT NULL AND NOT (extract(dow FROM now() AT TIME ZONE l.timezone)::integer=ANY(p.available_days)))
		           OR (p.available_until_time IS NOT NULL AND (now() AT TIME ZONE l.timezone)::time>p.available_until_time) THEN 0
		         WHEN p.quantity_control='portions' THEN GREATEST(COALESCE(pa.portion_quantity,0)-COALESCE(pa.sold_quantity,0),0)
		         WHEN p.quantity_control='inventory' THEN GREATEST(COALESCE(sb.quantity,0),0)
		       END::float8,
		       CASE WHEN p.quantity_control='inventory' THEN ii.unit END
		FROM products p
		JOIN locations l ON l.id=$7 AND l.organization_id=p.organization_id AND l.active
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN product_availability pa ON pa.organization_id=p.organization_id AND pa.location_id=l.id
		  AND pa.product_id=p.id AND pa.business_date=(now() AT TIME ZONE l.timezone)::date
		LEFT JOIN inventory_items ii ON ii.organization_id=p.organization_id AND ii.product_id=p.id AND ii.active
		LEFT JOIN stock_balances sb ON sb.organization_id=p.organization_id AND sb.location_id=l.id AND sb.inventory_item_id=ii.id
		WHERE p.organization_id=$1 AND (p.name ILIKE $2 OR p.sku ILIKE $2)
		  AND ($3='' OR p.category_id::text=$3)
		  AND ($4='' OR ($4='active' AND p.active) OR ($4='inactive' AND NOT p.active))
		  AND NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id)
		ORDER BY p.updated_at DESC,p.name LIMIT $5 OFFSET $6`, s.OrganizationID, search, categoryID, status, size, (page-1)*size, s.LocationID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.ProductListItem{}
	for rows.Next() {
		var p domain.ProductListItem
		if err = rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime, &p.ServiceDestination, &p.AvailableQuantity, &p.InventoryUnit); err != nil {
			return out, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Catalog) ProductCount(ctx context.Context, s scope) (int, error) {
	var count int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM products WHERE organization_id=$1`, s.OrganizationID).Scan(&count)
	return count, err
}
func (r *Catalog) HasCancellableOrderUsage(ctx context.Context, s scope, id string) (bool, error) {
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
			      SELECT 1
			      FROM order_item_combo_selections s
			      WHERE s.organization_id=oi.organization_id
			        AND s.order_item_id=oi.id
			        AND s.option_product_id=$2
			    )
			  )
		)`, s.OrganizationID, id).Scan(&used)
	return used, err
}
var likeEscaper = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)

// Allergens lists the shared catalog plus the organization's own entries,
// matching without case or accents.
func (r *Catalog) Allergens(ctx context.Context, s scope, query string) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT name FROM allergens
		WHERE (organization_id IS NULL OR organization_id=$1)
		  AND normalized_name LIKE '%' || normalize_allergen_name($2) || '%' ESCAPE '\'
		ORDER BY normalized_name LIMIT 20`, s.OrganizationID, likeEscaper.Replace(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	return items, rows.Err()
}
// CreateAllergen inserts only when no shared or own allergen has the same
// normalized name; the partial unique index covers concurrent inserts.
func (r *Catalog) CreateAllergen(ctx context.Context, s scope, name string) (domain.Allergen, error) {
	var out domain.Allergen
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO allergens(organization_id,name)
		SELECT $1,$2
		WHERE NOT EXISTS (
			SELECT 1 FROM allergens
			WHERE (organization_id IS NULL OR organization_id=$1)
			  AND normalized_name=normalize_allergen_name($2)
		)
		RETURNING id,name`, s.OrganizationID, name).Scan(&out.ID, &out.Name)
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return out, application.ErrAllergenConflict
	}
	return out, err
}
func (r *Catalog) SetImage(ctx context.Context, s scope, id, url string) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE products SET image_url=$3,updated_at=now() WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id, url)
	return err
}
