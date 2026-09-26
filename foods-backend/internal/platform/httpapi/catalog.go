package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
	Active       bool   `json:"active"`
	ProductScope string `json:"productScope"`
	ProductCount int    `json:"productCount"`
}
type categoryInput struct {
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
	Active       *bool  `json:"active,omitempty"`
	ProductScope string `json:"productScope"`
}
type product struct {
	ID                 string   `json:"id"`
	SKU                string   `json:"sku"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	CategoryID         *string  `json:"categoryId"`
	CategoryName       *string  `json:"categoryName"`
	Price              string   `json:"price"`
	Active             bool     `json:"active"`
	ProductType        string   `json:"productType"`
	QuantityControl    string   `json:"quantityControl"`
	ImageURL           *string  `json:"imageUrl"`
	PrepMinutes        *int     `json:"prepMinutes"`
	Allergens          []string `json:"allergens"`
	Featured           bool     `json:"featured"`
	CostPrice          *string  `json:"costPrice"`
	AvailableFrom      *string  `json:"availableFrom"`
	AvailableUntil     *string  `json:"availableUntil"`
	AvailableDays      []int    `json:"availableDays"`
	AvailableUntilTime *string  `json:"availableUntilTime"`
}
type productInput struct {
	SKU                string   `json:"sku"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	CategoryID         *string  `json:"categoryId"`
	Price              string   `json:"price"`
	Active             *bool    `json:"active,omitempty"`
	ProductType        string   `json:"productType"`
	QuantityControl    string   `json:"quantityControl"`
	ImageURL           *string  `json:"imageUrl"`
	PrepMinutes        *int     `json:"prepMinutes"`
	Allergens          []string `json:"allergens"`
	Featured           *bool    `json:"featured,omitempty"`
	CostPrice          *string  `json:"costPrice"`
	AvailableFrom      *string  `json:"availableFrom"`
	AvailableUntil     *string  `json:"availableUntil"`
	AvailableDays      []int    `json:"availableDays"`
	AvailableUntilTime *string  `json:"availableUntilTime"`
}

func pageParams(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func (a *API) audit(r *http.Request, action, entityType, entityID string) {
	s := r.Context().Value(scopeKey{}).(scope)
	_, _ = a.db.Exec(r.Context(), `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id) VALUES($1,$2,$3,$4,$5,$6)`, s.OrganizationID, s.LocationID, s.UserID, action, entityType, entityID)
}

// nextSKU genera un SKU secuencial legible (PROD-0001) por organización.
func (a *API) nextSKU(r *http.Request, orgID string) (string, error) {
	var count int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM products WHERE organization_id=$1`, orgID).Scan(&count); err != nil {
		return "", err
	}
	return fmt.Sprintf("PROD-%04d", count+1), nil
}

func normalizeCategoryProductScope(value, fallback string) (string, string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback, ""
	}
	switch value {
	case "prepared", "retail", "both":
		return value, ""
	default:
		return value, "El uso de la categoría no es válido."
	}
}

func (a *API) listCategories(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	includeInactive := r.URL.Query().Get("includeInactive") == "true"
	productType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("productType")))
	if productType != "" && productType != "prepared" && productType != "retail" {
		fail(w, 400, "invalid_category_filter", "El tipo de producto para filtrar categorías no es válido.")
		return
	}
	page, size := pageParams(r)
	var total int
	if err := a.db.QueryRow(r.Context(), `
		SELECT count(*)
		FROM menu_categories
		WHERE organization_id=$1
		  AND ($2 OR active)
		  AND ($3='' OR product_scope='both' OR product_scope=$3)`,
		s.OrganizationID, includeInactive, productType,
	).Scan(&total); err != nil {
		fail(w, 503, "categories_unavailable", "No pudimos cargar las categorías.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
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
		fail(w, 503, "categories_unavailable", "No pudimos cargar las categorías.")
		return
	}
	defer rows.Close()
	items := []category{}
	for rows.Next() {
		var c category
		if err = rows.Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active, &c.ProductScope, &c.ProductCount); err != nil {
			fail(w, 503, "categories_unavailable", "No pudimos cargar las categorías.")
			return
		}
		items = append(items, c)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}
func (a *API) createCategory(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in categoryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		fail(w, 400, "invalid_category", "El nombre de la categoría es obligatorio.")
		return
	}
	productScope, invalid := normalizeCategoryProductScope(in.ProductScope, "prepared")
	if invalid != "" {
		fail(w, 400, "invalid_category", invalid)
		return
	}
	var c category
	err := a.db.QueryRow(r.Context(), `
		INSERT INTO menu_categories(organization_id,name,sort_order,product_scope)
		VALUES($1,$2,$3,$4)
		RETURNING id,name,sort_order,active,product_scope`,
		s.OrganizationID, strings.TrimSpace(in.Name), in.SortOrder, productScope,
	).Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active, &c.ProductScope)
	if err != nil {
		fail(w, 409, "category_conflict", "Ya existe una categoría con ese nombre.")
		return
	}
	a.audit(r, "category.created", "menu_category", c.ID)
	writeJSON(w, 201, c)
}
func (a *API) updateCategory(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in categoryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
		fail(w, 400, "invalid_category", "El nombre de la categoría es obligatorio.")
		return
	}
	productScope, invalid := normalizeCategoryProductScope(in.ProductScope, "")
	if invalid != "" {
		fail(w, 400, "invalid_category", invalid)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	if productScope == "prepared" || productScope == "retail" {
		incompatibleType := "retail"
		if productScope == "retail" {
			incompatibleType = "prepared"
		}
		var incompatibleProducts int
		if err := a.db.QueryRow(r.Context(), `
			SELECT count(*)
			FROM products
			WHERE organization_id=$1 AND category_id=$2 AND product_type=$3`,
			s.OrganizationID, r.PathValue("id"), incompatibleType,
		).Scan(&incompatibleProducts); err != nil {
			fail(w, 503, "category_unavailable", "No pudimos validar los productos de la categoría.")
			return
		}
		if incompatibleProducts > 0 {
			fail(w, 409, "category_scope_in_use", "La categoría contiene productos de otro tipo. Usa «Ambos» o mueve esos productos antes de cambiar su uso.")
			return
		}
	}
	var c category
	err := a.db.QueryRow(r.Context(), `
		UPDATE menu_categories
		SET name=$3,sort_order=$4,active=$5,
		    product_scope=COALESCE(NULLIF($6,''),product_scope)
		WHERE id=$2 AND organization_id=$1
		RETURNING id,name,sort_order,active,product_scope`,
		s.OrganizationID, r.PathValue("id"), strings.TrimSpace(in.Name), in.SortOrder, active, productScope,
	).Scan(&c.ID, &c.Name, &c.SortOrder, &c.Active, &c.ProductScope)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "category_not_found", "La categoría no existe.")
		return
	}
	if err != nil {
		fail(w, 409, "category_conflict", "No se pudo actualizar la categoría.")
		return
	}
	a.audit(r, "category.updated", "menu_category", c.ID)
	writeJSON(w, 200, c)
}
func (a *API) deactivateCategory(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	id := r.PathValue("id")
	var products int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM products WHERE organization_id=$1 AND category_id=$2 AND active`, s.OrganizationID, id).Scan(&products); err != nil {
		fail(w, 503, "category_unavailable", "No pudimos validar la categoría.")
		return
	}
	if products > 0 {
		fail(w, 409, "category_in_use", "Desactiva o mueve sus productos antes de desactivar la categoría.")
		return
	}
	tag, err := a.db.Exec(r.Context(), `UPDATE menu_categories SET active=false WHERE organization_id=$1 AND id=$2 AND active`, s.OrganizationID, id)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 404, "category_not_found", "La categoría no existe o ya está inactiva.")
		return
	}
	a.audit(r, "category.deactivated", "menu_category", id)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listProducts(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	status := r.URL.Query().Get("status")
	categoryID := r.URL.Query().Get("categoryId")
	var total int
	err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM products p WHERE p.organization_id=$1 AND (p.name ILIKE $2 OR p.sku ILIKE $2) AND ($3='' OR p.category_id::text=$3) AND ($4='' OR ($4='active' AND p.active) OR ($4='inactive' AND NOT p.active)) AND NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id)`, s.OrganizationID, search, categoryID, status).Scan(&total)
	if err != nil {
		fail(w, 503, "products_unavailable", "No pudimos cargar los productos.")
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT p.id,p.sku,p.name,p.description,p.category_id,c.name,p.price::text,p.active,p.product_type,p.quantity_control,p.image_url,p.prep_minutes,p.allergens,p.featured,p.cost_price::text,p.available_from::text,p.available_until::text,p.available_days,p.available_until_time::text FROM products p LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id WHERE p.organization_id=$1 AND (p.name ILIKE $2 OR p.sku ILIKE $2) AND ($3='' OR p.category_id::text=$3) AND ($4='' OR ($4='active' AND p.active) OR ($4='inactive' AND NOT p.active)) AND NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id) ORDER BY p.updated_at DESC,p.name LIMIT $5 OFFSET $6`, s.OrganizationID, search, categoryID, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "products_unavailable", "No pudimos cargar los productos.")
		return
	}
	defer rows.Close()
	items := []product{}
	for rows.Next() {
		var p product
		if err = rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime); err != nil {
			fail(w, 503, "products_unavailable", "No pudimos cargar los productos.")
			return
		}
		items = append(items, p)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}
func (a *API) getProduct(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var p product
	err := a.db.QueryRow(r.Context(), `SELECT p.id,p.sku,p.name,p.description,p.category_id,c.name,p.price::text,p.active,p.product_type,p.quantity_control,p.image_url,p.prep_minutes,p.allergens,p.featured,p.cost_price::text,p.available_from::text,p.available_until::text,p.available_days,p.available_until_time::text FROM products p LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id WHERE p.organization_id=$1 AND p.id=$2`, s.OrganizationID, r.PathValue("id")).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.CategoryName, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "product_unavailable", "No pudimos cargar el producto.")
		return
	}
	writeJSON(w, 200, p)
}

// normalizeProduct valida únicamente datos comerciales y la clasificación
// de cantidad. Las cantidades nunca se guardan en products.
func normalizeProduct(in productInput) (productInput, string) {
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Price) == "" {
		return in, "Nombre y precio son obligatorios."
	}
	in.ProductType = strings.ToLower(strings.TrimSpace(in.ProductType))
	if in.ProductType == "" {
		in.ProductType = "prepared"
	}
	switch in.ProductType {
	case "prepared", "retail":
	default:
		return in, "El tipo de producto no es válido."
	}
	if in.QuantityControl == "" {
		in.QuantityControl = "none"
	}
	switch in.QuantityControl {
	case "none", "portions", "inventory":
	default:
		return in, "El control de cantidad no es válido."
	}
	return in, ""
}
func (a *API) createProduct(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in productInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_product", "Nombre y precio son obligatorios.")
		return
	}
	in, invalid := normalizeProduct(in)
	if invalid != "" {
		fail(w, 400, "invalid_product", invalid)
		return
	}
	if strings.TrimSpace(in.SKU) == "" {
		generated, err := a.nextSKU(r, s.OrganizationID)
		if err != nil {
			fail(w, 503, "sku_unavailable", "No pudimos generar un código de producto.")
			return
		}
		in.SKU = generated
	}
	createFeatured := false
	if in.Featured != nil {
		createFeatured = *in.Featured
	}
	var p product
	err := a.db.QueryRow(r.Context(), `INSERT INTO products(organization_id,category_id,sku,name,description,price,active,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price,available_from,available_until,available_days,available_until_time) VALUES($1,$2,$3,$4,$5,$6,true,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) RETURNING id,sku,name,description,category_id,price::text,active,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price::text,available_from::text,available_until::text,available_days,available_until_time::text`, s.OrganizationID, in.CategoryID, strings.TrimSpace(in.SKU), strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, in.ProductType, in.QuantityControl, in.ImageURL, in.PrepMinutes, in.Allergens, createFeatured, in.CostPrice, in.AvailableFrom, in.AvailableUntil, in.AvailableDays, in.AvailableUntilTime).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime)
	if err != nil {
		fail(w, 409, "product_conflict", "Revisa la categoría y el precio.")
		return
	}
	a.audit(r, "product.created", "product", p.ID)
	writeJSON(w, 201, p)
}
func (a *API) updateProduct(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in productInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_product", "Nombre y precio son obligatorios.")
		return
	}
	productTypeProvided := strings.TrimSpace(in.ProductType) != ""
	in, invalid := normalizeProduct(in)
	if invalid != "" {
		fail(w, 400, "invalid_product", invalid)
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	featured := false
	if in.Featured != nil {
		featured = *in.Featured
	}
	skuValue := strings.TrimSpace(in.SKU)
	if skuValue == "" {
		skuValue = r.PathValue("id")
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "product_unavailable", "No pudimos actualizar el producto.")
		return
	}
	defer tx.Rollback(r.Context())

	var currentControl, currentProductType string
	err = tx.QueryRow(r.Context(), `
		SELECT quantity_control,product_type
		FROM products
		WHERE organization_id=$1 AND id=$2
		FOR UPDATE`, s.OrganizationID, r.PathValue("id")).Scan(&currentControl, &currentProductType)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "product_not_found", "El producto no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "product_unavailable", "No pudimos validar el producto.")
		return
	}
	if !productTypeProvided {
		in.ProductType = currentProductType
	}
	if currentControl != in.QuantityControl {
		if currentControl == "none" && in.QuantityControl != "none" {
			usedByOpenOrder, usageErr := productHasCancellableOrderUsage(r.Context(), tx, s.OrganizationID, r.PathValue("id"))
			if usageErr != nil {
				fail(w, 503, "product_unavailable", "No pudimos validar los pedidos abiertos del producto.")
				return
			}
			if usedByOpenOrder {
				fail(w, 409, "quantity_control_open_orders", "Cierra o cancela los pedidos abiertos de este producto antes de activar un control de cantidad.")
				return
			}
		}
		var used bool
		switch currentControl {
		case "inventory":
			err = tx.QueryRow(r.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM stock_movements
					WHERE organization_id=$1 AND product_id=$2
				)`, s.OrganizationID, r.PathValue("id")).Scan(&used)
		case "portions":
			err = tx.QueryRow(r.Context(), `
				SELECT EXISTS(
					SELECT 1 FROM product_availability
					WHERE organization_id=$1 AND product_id=$2 AND sold_quantity>0
				)`, s.OrganizationID, r.PathValue("id")).Scan(&used)
		}
		if err != nil {
			fail(w, 503, "product_unavailable", "No pudimos validar el historial de cantidades.")
			return
		}
		if used {
			fail(w, 409, "quantity_control_in_use", "No puedes cambiar el control de cantidad porque el producto ya tiene movimientos o ventas asociados.")
			return
		}
	}

	var p product
	err = tx.QueryRow(r.Context(), `UPDATE products SET category_id=$3,sku=$4,name=$5,description=$6,price=$7,active=$8,product_type=$9,quantity_control=$10,image_url=$11,prep_minutes=$12,allergens=$13,featured=$14,cost_price=$15,available_from=$16,available_until=$17,available_days=$18,available_until_time=$19,updated_at=now() WHERE organization_id=$1 AND id=$2 RETURNING id,sku,name,description,category_id,price::text,active,product_type,quantity_control,image_url,prep_minutes,allergens,featured,cost_price::text,available_from::text,available_until::text,available_days,available_until_time::text`, s.OrganizationID, r.PathValue("id"), in.CategoryID, skuValue, strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), in.Price, active, in.ProductType, in.QuantityControl, in.ImageURL, in.PrepMinutes, in.Allergens, featured, in.CostPrice, in.AvailableFrom, in.AvailableUntil, in.AvailableDays, in.AvailableUntilTime).Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.CategoryID, &p.Price, &p.Active, &p.ProductType, &p.QuantityControl, &p.ImageURL, &p.PrepMinutes, &p.Allergens, &p.Featured, &p.CostPrice, &p.AvailableFrom, &p.AvailableUntil, &p.AvailableDays, &p.AvailableUntilTime)
	if err != nil {
		fail(w, 409, "product_conflict", "No se pudo actualizar el producto.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "product_unavailable", "No pudimos actualizar el producto.")
		return
	}
	a.audit(r, "product.updated", "product", p.ID)
	writeJSON(w, 200, p)
}
func (a *API) deactivateProduct(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	id := r.PathValue("id")
	tag, err := a.db.Exec(r.Context(), `UPDATE products SET active=false,updated_at=now() WHERE organization_id=$1 AND id=$2 AND active`, s.OrganizationID, id)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 404, "product_not_found", "El producto no existe o ya está inactivo.")
		return
	}
	a.audit(r, "product.deactivated", "product", id)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) uploadProductImage(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	id := r.PathValue("id")
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		fail(w, 400, "image_too_large", "La imagen no puede pesar más de 5 MB.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "image_required", "Selecciona una imagen para subir.")
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	default:
		fail(w, 400, "image_format", "Solo se aceptan imágenes PNG, JPEG o WebP.")
		return
	}
	if err := os.MkdirAll("./uploads/products", 0755); err != nil {
		fail(w, 503, "image_storage", "No pudimos guardar la imagen.")
		return
	}
	filename := id + ext
	dst, err := os.Create("./uploads/products/" + filename)
	if err != nil {
		fail(w, 503, "image_storage", "No pudimos guardar la imagen.")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		fail(w, 503, "image_storage", "No pudimos guardar la imagen.")
		return
	}
	imageURL := "/uploads/products/" + filename
	_, err = a.db.Exec(r.Context(), `UPDATE products SET image_url=$3,updated_at=now() WHERE organization_id=$1 AND id=$2`, s.OrganizationID, id, imageURL)
	if err != nil {
		fail(w, 503, "image_storage", "No pudimos vincular la imagen al producto.")
		return
	}
	a.audit(r, "product.image_uploaded", "product", id)
	writeJSON(w, 200, map[string]string{"imageUrl": imageURL})
}

func (a *API) listAllergens(w http.ResponseWriter, r *http.Request) {
	q := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	rows, err := a.db.Query(r.Context(), `SELECT name FROM allergens WHERE name ILIKE $1 ORDER BY name LIMIT 20`, q)
	if err != nil {
		fail(w, 503, "allergens_unavailable", "No pudimos cargar los alérgenos.")
		return
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			fail(w, 503, "allergens_unavailable", "No pudimos cargar los alérgenos.")
			return
		}
		items = append(items, name)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
