package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type inventoryItemView struct {
	InventoryItemID string  `json:"inventoryItemId"`
	ProductID       *string `json:"productId"`
	SKU             string  `json:"sku"`
	Name            string  `json:"name"`
	Kind            string  `json:"kind"`
	CategoryName    *string `json:"categoryName"`
	Active          bool    `json:"active"`
	Unit            string  `json:"unit"`
	Quantity        string  `json:"quantity"`
	MinimumStock    string  `json:"minimumStock"`
	Status          string  `json:"status"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type inventoryPresentationOption struct {
	ID                   string `json:"id"`
	PresentationType     string `json:"presentationType"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
}

type inventoryProductOption struct {
	ID              string                        `json:"id"`
	ProductID       *string                       `json:"productId"`
	SKU             string                        `json:"sku"`
	Name            string                        `json:"name"`
	Kind            string                        `json:"kind"`
	CategoryName    *string                       `json:"categoryName"`
	QuantityControl *string                       `json:"quantityControl"`
	Unit            string                        `json:"unit"`
	Quantity        string                        `json:"quantity"`
	MinimumStock    string                        `json:"minimumStock"`
	Presentations   []inventoryPresentationOption `json:"presentations"`
}

type inventoryNewProductInput struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CategoryID  *string `json:"categoryId"`
	Price       string  `json:"price"`
}

type inventoryNewIngredientInput struct {
	Name string `json:"name"`
}

type inventoryEntryInput struct {
	InventoryItemID       string                       `json:"inventoryItemId"`
	ProductID             string                       `json:"productId"`
	NewProduct            *inventoryNewProductInput    `json:"newProduct"`
	NewIngredient         *inventoryNewIngredientInput `json:"newIngredient"`
	Quantity              float64                      `json:"quantity"`
	Unit                  string                       `json:"unit"`
	PresentationType      string                       `json:"presentationType"`
	UnitsPerPresentation  float64                      `json:"unitsPerPresentation"`
	MinimumStock          float64                      `json:"minimumStock"`
	Note                  string                       `json:"note"`
}

type inventoryCatalogCreateResult struct {
	InventoryItemID string
	ProductID       *string
	SKU             string
	Name            string
	Unit            string
	Kind            string
}

type inventoryCatalogCreateError struct {
	Status  int
	Code    string
	Message string
}

type inventoryMovementView struct {
	ID              string  `json:"id"`
	InventoryItemID string  `json:"inventoryItemId"`
	ProductID       *string `json:"productId"`
	ItemName        string  `json:"itemName"`
	MovementType    string  `json:"movementType"`
	AdjustmentType  *string `json:"adjustmentType,omitempty"`
	Reason          *string `json:"reason,omitempty"`
	QuantityDelta   string  `json:"quantityDelta"`
	BalanceBefore   string  `json:"balanceBefore"`
	BalanceAfter    string  `json:"balanceAfter"`
	SourceType      string  `json:"sourceType"`
	SourceID        string  `json:"sourceId"`
	SourceReference string  `json:"sourceReference"`
	Note            string  `json:"note"`
	CreatedByName   string  `json:"createdByName"`
	CreatedAt       time.Time `json:"createdAt"`
}

type inventoryAdjustmentInput struct {
	InventoryItemID string  `json:"inventoryItemId"`
	MovementType    string  `json:"movementType"`
	Reason          string  `json:"reason"`
	Quantity        float64 `json:"quantity"`
	Observation     string  `json:"observation"`
}

func productHasCancellableOrderUsage(ctx context.Context, tx pgx.Tx, organizationID, productID string) (bool, error) {
	var used bool
	err := tx.QueryRow(ctx, `
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
		)`, organizationID, productID).Scan(&used)
	return used, err
}

func normalizeInventoryEntry(in inventoryEntryInput) (inventoryEntryInput, string) {
	in.InventoryItemID = strings.TrimSpace(in.InventoryItemID)
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.Unit = strings.TrimSpace(in.Unit)
	in.PresentationType = strings.ToLower(strings.TrimSpace(in.PresentationType))
	in.Note = strings.TrimSpace(in.Note)
	if in.Quantity <= 0 {
		return in, "La cantidad de entrada debe ser mayor que cero."
	}
	in.Quantity = math.Round(in.Quantity*1000) / 1000
	in.UnitsPerPresentation = math.Round(in.UnitsPerPresentation*1000) / 1000
	if in.Quantity <= 0 {
		return in, "La cantidad de entrada debe ser al menos 0.001."
	}
	if in.MinimumStock < 0 {
		return in, "El stock mínimo no puede ser negativo."
	}
	if in.Unit == "" {
		in.Unit = "und"
	}
	if in.PresentationType == "" {
		in.PresentationType = "unit"
	}
	switch in.PresentationType {
	case "unit":
		in.UnitsPerPresentation = 1
	case "package", "box":
		if math.Abs(in.Quantity-math.Round(in.Quantity)) > 0.000001 {
			return in, "La cantidad de paquetes o cajas debe ser un número entero. Para fracciones, registra unidades sueltas."
		}
		if in.UnitsPerPresentation <= 1 {
			return in, "Indica cuántas unidades base contiene cada paquete o caja."
		}
	default:
		return in, "La presentación de entrada no es válida."
	}

	sourceCount := 0
	if in.InventoryItemID != "" {
		sourceCount++
	}
	if in.ProductID != "" {
		sourceCount++
	}
	if in.NewProduct != nil {
		sourceCount++
	}
	if in.NewIngredient != nil {
		sourceCount++
	}
	if sourceCount != 1 {
		return in, "Selecciona un artículo existente, crea un producto vendible o crea un insumo."
	}

	if in.NewProduct != nil {
		in.NewProduct.Name = strings.TrimSpace(in.NewProduct.Name)
		in.NewProduct.SKU = strings.TrimSpace(in.NewProduct.SKU)
		in.NewProduct.Description = strings.TrimSpace(in.NewProduct.Description)
		in.NewProduct.Price = strings.TrimSpace(in.NewProduct.Price)
		if in.NewProduct.Name == "" || in.NewProduct.Price == "" {
			return in, "El nuevo producto vendible necesita nombre y precio de venta."
		}
		if in.NewProduct.CategoryID == nil || strings.TrimSpace(*in.NewProduct.CategoryID) == "" {
			return in, "Selecciona una categoría para la mercadería vendible."
		}
		categoryID := strings.TrimSpace(*in.NewProduct.CategoryID)
		in.NewProduct.CategoryID = &categoryID
	}
	if in.NewIngredient != nil {
		in.NewIngredient.Name = strings.TrimSpace(in.NewIngredient.Name)
		if in.NewIngredient.Name == "" {
			return in, "El nuevo insumo necesita un nombre."
		}
	}
	return in, ""
}

func createInventoryCatalogItem(ctx context.Context, tx pgx.Tx, organizationID string, in inventoryEntryInput) (inventoryCatalogCreateResult, *inventoryCatalogCreateError) {
	result := inventoryCatalogCreateResult{Unit: in.Unit}
	switch {
	case in.NewProduct != nil:
		productIn := productInput{
			SKU:             in.NewProduct.SKU,
			Name:            in.NewProduct.Name,
			Description:     in.NewProduct.Description,
			CategoryID:      in.NewProduct.CategoryID,
			Price:           in.NewProduct.Price,
			ProductType:     "retail",
			QuantityControl: "inventory",
		}
		if _, invalidProduct := normalizeProduct(productIn); invalidProduct != "" {
			return result, &inventoryCatalogCreateError{Status: 400, Code: "invalid_product", Message: invalidProduct}
		}
		var categoryAllowed bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM menu_categories
				WHERE id=$1 AND organization_id=$2 AND active
				  AND product_scope IN ('retail','both')
			)`,
			in.NewProduct.CategoryID, organizationID,
		).Scan(&categoryAllowed); err != nil {
			return result, &inventoryCatalogCreateError{Status: 503, Code: "categories_unavailable", Message: "No pudimos validar la categoría."}
		}
		if !categoryAllowed {
			return result, &inventoryCatalogCreateError{Status: 409, Code: "category_not_retail", Message: "La categoría seleccionada no admite mercadería vendible."}
		}
		var productID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO products(organization_id,category_id,sku,name,description,price,active,product_type,quantity_control)
			VALUES($1,$2,COALESCE(NULLIF($3,''),'PROD-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,8))),$4,$5,$6,true,'retail','inventory')
			RETURNING id,sku,name`,
			organizationID, in.NewProduct.CategoryID, in.NewProduct.SKU,
			in.NewProduct.Name, in.NewProduct.Description, in.NewProduct.Price,
		).Scan(&productID, &result.SKU, &result.Name); err != nil {
			return result, &inventoryCatalogCreateError{Status: 409, Code: "product_conflict", Message: "No pudimos crear el producto. Revisa nombre, categoría y precio."}
		}
		result.ProductID = &productID
		result.Kind = "product"
		if err := tx.QueryRow(ctx, `
			INSERT INTO inventory_items(
				organization_id,sku,name,unit,minimum_stock,active,product_id
			)
			VALUES($1,'SELL-'||upper(substr(replace($2::text,'-',''),1,12)),$3,$4,$5,true,$2::uuid)
			RETURNING id,unit`,
			organizationID, productID, result.Name, in.Unit, in.MinimumStock,
		).Scan(&result.InventoryItemID, &result.Unit); err != nil {
			return result, &inventoryCatalogCreateError{Status: 503, Code: "inventory_unavailable", Message: "No pudimos vincular el producto al inventario."}
		}
	case in.NewIngredient != nil:
		result.Name = in.NewIngredient.Name
		result.Kind = "ingredient"
		if err := tx.QueryRow(ctx, `
			INSERT INTO inventory_items(
				organization_id,sku,name,unit,minimum_stock,active
			)
			VALUES(
				$1,
				'ING-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,10)),
				$2,$3,$4,true
			)
			RETURNING id,sku,unit`,
			organizationID, result.Name, in.Unit, in.MinimumStock,
		).Scan(&result.InventoryItemID, &result.SKU, &result.Unit); err != nil {
			return result, &inventoryCatalogCreateError{Status: 409, Code: "ingredient_conflict", Message: "No pudimos crear el insumo."}
		}
	default:
		return result, &inventoryCatalogCreateError{Status: 400, Code: "invalid_inventory_item", Message: "Selecciona el tipo de artículo a crear."}
	}
	return result, nil
}

func ensureInventoryPresentation(ctx context.Context, tx pgx.Tx, organizationID, inventoryItemID, presentationType string, unitsPerPresentation float64) (string, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory_presentations(
			organization_id,inventory_item_id,presentation_type,units_per_presentation,active
		)
		VALUES($1,$2,'unit',1,true)
		ON CONFLICT (organization_id,inventory_item_id,presentation_type,units_per_presentation)
		DO UPDATE SET active=true,updated_at=now()`,
		organizationID, inventoryItemID); err != nil {
		return "", err
	}
	var presentationID string
	err := tx.QueryRow(ctx, `
		INSERT INTO inventory_presentations(
			organization_id,inventory_item_id,presentation_type,units_per_presentation,active
		)
		VALUES($1,$2,$3,$4,true)
		ON CONFLICT (organization_id,inventory_item_id,presentation_type,units_per_presentation)
		DO UPDATE SET active=true,updated_at=now()
		RETURNING id`,
		organizationID, inventoryItemID, presentationType, unitsPerPresentation).
		Scan(&presentationID)
	return presentationID, err
}

func (a *API) listInventory(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	var total int
	if err := a.db.QueryRow(r.Context(), `
		SELECT count(*)
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE ii.organization_id=$1 AND ii.active
		  AND (COALESCE(p.name,ii.name) ILIKE $2 OR COALESCE(p.sku,ii.sku) ILIKE $2)`,
		s.OrganizationID, search).Scan(&total); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos cargar el inventario.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT ii.id,p.id,COALESCE(p.sku,ii.sku),COALESCE(p.name,ii.name),
		       CASE WHEN p.id IS NULL THEN 'ingredient' ELSE 'product' END,
		       c.name,COALESCE(p.active,ii.active),ii.unit,
		       COALESCE(sb.quantity,0)::text,ii.minimum_stock::text,
		       CASE
		         WHEN COALESCE(sb.quantity,0)<=0 THEN 'out'
		         WHEN ii.minimum_stock>0 AND COALESCE(sb.quantity,0)<=ii.minimum_stock THEN 'low'
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
		WHERE ii.organization_id=$1 AND ii.active
		  AND (COALESCE(p.name,ii.name) ILIKE $3 OR COALESCE(p.sku,ii.sku) ILIKE $3)
		ORDER BY COALESCE(p.active,ii.active) DESC,COALESCE(p.name,ii.name)
		LIMIT $4 OFFSET $5`,
		s.OrganizationID, s.LocationID, search, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos cargar el inventario.")
		return
	}
	defer rows.Close()
	items := []inventoryItemView{}
	for rows.Next() {
		var item inventoryItemView
		if err := rows.Scan(
			&item.InventoryItemID, &item.ProductID, &item.SKU, &item.Name, &item.Kind,
			&item.CategoryName, &item.Active, &item.Unit, &item.Quantity, &item.MinimumStock,
			&item.Status, &item.UpdatedAt,
		); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos cargar el inventario.")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos cargar el inventario.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) listInventoryProducts(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	rows, err := a.db.Query(r.Context(), `
		SELECT ii.id,p.id,COALESCE(p.sku,ii.sku),COALESCE(p.name,ii.name),
		       CASE WHEN p.id IS NULL THEN 'ingredient' ELSE 'product' END,
		       c.name,p.quantity_control,ii.unit,COALESCE(sb.quantity,0)::text,ii.minimum_stock::text,
		       COALESCE((
		         SELECT jsonb_agg(
		           jsonb_build_object(
		             'id',ip.id,
		             'presentationType',ip.presentation_type,
		             'unitsPerPresentation',ip.units_per_presentation::text
		           )
		           ORDER BY
		             CASE ip.presentation_type WHEN 'unit' THEN 0 WHEN 'package' THEN 1 ELSE 2 END,
		             ip.units_per_presentation
		         )
		         FROM inventory_presentations ip
		         WHERE ip.organization_id=ii.organization_id
		           AND ip.inventory_item_id=ii.id
		           AND ip.active
		       ),'[]'::jsonb)
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		LEFT JOIN menu_categories c
		  ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN stock_balances sb
		  ON sb.organization_id=ii.organization_id
		 AND sb.location_id=$2
		 AND sb.inventory_item_id=ii.id
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
		  AND (COALESCE(p.name,ii.name) ILIKE $3 OR COALESCE(p.sku,ii.sku) ILIKE $3)
		ORDER BY COALESCE(p.name,ii.name)
		LIMIT 100`, s.OrganizationID, s.LocationID, search)
	if err != nil {
		fail(w, 503, "inventory_products_unavailable", "No pudimos cargar los artículos de inventario.")
		return
	}
	defer rows.Close()
	items := []inventoryProductOption{}
	for rows.Next() {
		var item inventoryProductOption
		var presentationsJSON []byte
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.SKU, &item.Name, &item.Kind, &item.CategoryName,
			&item.QuantityControl, &item.Unit, &item.Quantity, &item.MinimumStock, &presentationsJSON,
		); err != nil {
			fail(w, 503, "inventory_products_unavailable", "No pudimos cargar los artículos de inventario.")
			return
		}
		if err := json.Unmarshal(presentationsJSON, &item.Presentations); err != nil {
			fail(w, 503, "inventory_products_unavailable", "No pudimos cargar las presentaciones del artículo.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (a *API) createInventoryEntry(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in inventoryEntryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_inventory_entry", "Revisa los datos de la entrada.")
		return
	}
	var invalid string
	in, invalid = normalizeInventoryEntry(in)
	if invalid != "" {
		fail(w, 400, "invalid_inventory_entry", invalid)
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar la entrada.")
		return
	}
	defer tx.Rollback(r.Context())

	var productID *string
	var itemSKU, itemName, inventoryItemID, inventoryUnit string
	createdProduct := false
	createdInventoryItem := false

	switch {
	case in.NewProduct != nil, in.NewIngredient != nil:
		created, createErr := createInventoryCatalogItem(r.Context(), tx, s.OrganizationID, in)
		if createErr != nil {
			fail(w, createErr.Status, createErr.Code, createErr.Message)
			return
		}
		inventoryItemID = created.InventoryItemID
		productID = created.ProductID
		itemSKU = created.SKU
		itemName = created.Name
		inventoryUnit = created.Unit
		createdProduct = created.Kind == "product"
		createdInventoryItem = true

	case in.InventoryItemID != "":
		var linkedProductID *string
		var quantityControl *string
		err = tx.QueryRow(r.Context(), `
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
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "inventory_item_not_found", "El artículo de inventario no existe o está inactivo.")
			return
		}
		if err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos validar el artículo de inventario.")
			return
		}
		if linkedProductID != nil {
			if quantityControl == nil || *quantityControl != "inventory" {
				fail(w, 409, "quantity_control_conflict", "El producto ya no está configurado como Inventario físico.")
				return
			}
			productID = linkedProductID
		}
		if _, err = tx.Exec(r.Context(), `
			UPDATE inventory_items
			SET minimum_stock=$3,updated_at=now()
			WHERE id=$1 AND organization_id=$2`,
			inventoryItemID, s.OrganizationID, in.MinimumStock); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos actualizar el stock mínimo.")
			return
		}

	case in.ProductID != "":
		var quantityControl string
		var existingProductID string
		err = tx.QueryRow(r.Context(), `
			SELECT id,sku,name,quantity_control
			FROM products
			WHERE id=$1 AND organization_id=$2 AND active
			FOR UPDATE`, in.ProductID, s.OrganizationID).
			Scan(&existingProductID, &itemSKU, &itemName, &quantityControl)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "product_not_found", "El producto no existe o está inactivo.")
			return
		}
		if err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos validar el producto.")
			return
		}
		if quantityControl != "inventory" {
			fail(w, 409, "quantity_control_conflict", "Solo los productos de Inventario físico pueden recibir entradas.")
			return
		}
		productID = &existingProductID
		err = tx.QueryRow(r.Context(), `
			INSERT INTO inventory_items(
				organization_id,sku,name,unit,minimum_stock,active,product_id
			)
			VALUES($1,'SELL-'||upper(substr(replace($2::text,'-',''),1,12)),$3,$4,$5,true,$2::uuid)
			ON CONFLICT (organization_id,product_id) WHERE product_id IS NOT NULL
			DO UPDATE SET
				name=EXCLUDED.name,
				minimum_stock=EXCLUDED.minimum_stock,
				active=true,
				updated_at=now()
			RETURNING id,unit`,
			s.OrganizationID, existingProductID, itemName, in.Unit, in.MinimumStock,
		).Scan(&inventoryItemID, &inventoryUnit)
		if err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos vincular el producto al inventario.")
			return
		}
	}

	if inventoryUnit != in.Unit {
		fail(w, 409, "inventory_unit_conflict", "La unidad base de este artículo es "+inventoryUnit+". Registra la entrada usando esa misma unidad base.")
		return
	}

	presentationID, err := ensureInventoryPresentation(
		r.Context(), tx, s.OrganizationID, inventoryItemID, in.PresentationType, in.UnitsPerPresentation,
	)
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos guardar la presentación de entrada.")
		return
	}

	stockQuantity := math.Round(in.Quantity*in.UnitsPerPresentation*1000) / 1000
	if stockQuantity <= 0 {
		fail(w, 400, "invalid_inventory_entry", "La equivalencia en unidades base debe ser mayor que cero.")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
		VALUES($1,$2,$3,0)
		ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
		s.OrganizationID, s.LocationID, inventoryItemID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos preparar el saldo de inventario.")
		return
	}

	var current float64
	if err = tx.QueryRow(r.Context(), `
		SELECT quantity::float8
		FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
		FOR UPDATE`, s.OrganizationID, s.LocationID, inventoryItemID).Scan(&current); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos bloquear el saldo de inventario.")
		return
	}
	balanceAfter := current + stockQuantity
	if _, err = tx.Exec(r.Context(), `
		UPDATE stock_balances
		SET quantity=$4,updated_at=now()
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, inventoryItemID, balanceAfter); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos actualizar el saldo.")
		return
	}

	var entryID string
	if err = tx.QueryRow(r.Context(), `
		INSERT INTO inventory_entries(
			organization_id,location_id,product_id,inventory_item_id,
			quantity,unit,presentation_id,presentation_type,units_per_presentation,note,created_by
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		s.OrganizationID, s.LocationID, productID, inventoryItemID,
		in.Quantity, in.Unit, presentationID, in.PresentationType,
		in.UnitsPerPresentation, in.Note, s.UserID).Scan(&entryID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el documento de entrada.")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		INSERT INTO stock_movements(
			organization_id,location_id,product_id,inventory_item_id,
			movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by
		)
		VALUES($1,$2,$3,$4,'entry',$5,$6,'inventory_entry',$7,$8,$9)`,
		s.OrganizationID, s.LocationID, productID, inventoryItemID,
		stockQuantity, balanceAfter, entryID, in.Note, s.UserID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar la entrada.")
		return
	}

	if createdProduct && productID != nil {
		a.audit(r, "product.created_from_inventory", "product", *productID)
	}
	if createdInventoryItem && productID == nil {
		a.audit(r, "inventory.ingredient_created", "inventory_item", inventoryItemID)
	}
	a.audit(r, "inventory.entry_created", "inventory_item", inventoryItemID)

	kind := "ingredient"
	if productID != nil {
		kind = "product"
	}
	writeJSON(w, 201, map[string]any{
		"id": entryID,
		"inventoryItemId": inventoryItemID,
		"productId": productID,
		"sku": itemSKU,
		"name": itemName,
		"kind": kind,
		"quantity": in.Quantity,
		"presentationId": presentationID,
		"presentationType": in.PresentationType,
		"unitsPerPresentation": in.UnitsPerPresentation,
		"stockQuantity": stockQuantity,
		"unit": in.Unit,
		"balance": balanceAfter,
		"createdProduct": createdProduct,
		"createdInventoryItem": createdInventoryItem,
	})
}

func validInventoryAdjustmentReason(movementType, reason string) bool {
	switch movementType {
	case "entry":
		return reason == "surplus_adjustment"
	case "exit":
		return reason == "shortage_adjustment" || reason == "waste" || reason == "expiration" || reason == "other_exit"
	default:
		return false
	}
}

func normalizeInventoryAdjustment(in inventoryAdjustmentInput) (inventoryAdjustmentInput, string) {
	in.InventoryItemID = strings.TrimSpace(in.InventoryItemID)
	in.MovementType = strings.ToLower(strings.TrimSpace(in.MovementType))
	in.Reason = strings.ToLower(strings.TrimSpace(in.Reason))
	in.Observation = strings.TrimSpace(in.Observation)
	in.Quantity = math.Round(in.Quantity*1000) / 1000
	if in.InventoryItemID == "" {
		return in, "Selecciona un artículo existente."
	}
	if in.MovementType != "entry" && in.MovementType != "exit" {
		return in, "Selecciona si el ajuste es una entrada o una salida."
	}
	if !validInventoryAdjustmentReason(in.MovementType, in.Reason) {
		return in, "Selecciona un motivo válido para el tipo de movimiento."
	}
	if in.Quantity <= 0 {
		return in, "La cantidad debe ser mayor que cero."
	}
	if len(in.Observation) > 240 {
		return in, "La observación no puede superar 240 caracteres."
	}
	return in, ""
}

func (a *API) createInventoryAdjustment(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in inventoryAdjustmentInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_inventory_adjustment", "Revisa los datos del ajuste.")
		return
	}
	var invalid string
	in, invalid = normalizeInventoryAdjustment(in)
	if invalid != "" {
		fail(w, 400, "invalid_inventory_adjustment", invalid)
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el ajuste.")
		return
	}
	defer tx.Rollback(r.Context())

	var productID *string
	var itemName, unit string
	err = tx.QueryRow(r.Context(), `
		SELECT ii.product_id,COALESCE(p.name,ii.name),ii.unit
		FROM inventory_items ii
		LEFT JOIN products p
		  ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE ii.id=$1 AND ii.organization_id=$2 AND ii.active
		  AND (p.id IS NULL OR p.active)
		FOR UPDATE OF ii`, in.InventoryItemID, s.OrganizationID).
		Scan(&productID, &itemName, &unit)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "inventory_item_not_found", "El artículo no existe o está inactivo.")
		return
	}
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos validar el artículo.")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
		VALUES($1,$2,$3,0)
		ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
		s.OrganizationID, s.LocationID, in.InventoryItemID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos preparar el saldo del artículo.")
		return
	}

	var stockBefore float64
	if err = tx.QueryRow(r.Context(), `
		SELECT quantity::float8
		FROM stock_balances
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
		FOR UPDATE`, s.OrganizationID, s.LocationID, in.InventoryItemID).Scan(&stockBefore); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos bloquear el saldo del artículo.")
		return
	}

	delta := in.Quantity
	if in.MovementType == "exit" {
		delta = -in.Quantity
	}
	stockAfter := math.Round((stockBefore+delta)*1000) / 1000
	if stockAfter < -0.000001 {
		fail(w, 409, "insufficient_stock", "La salida supera el stock actual del artículo.")
		return
	}
	if stockAfter < 0 {
		stockAfter = 0
	}

	if _, err = tx.Exec(r.Context(), `
		UPDATE stock_balances
		SET quantity=$4,updated_at=now()
		WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
		s.OrganizationID, s.LocationID, in.InventoryItemID, stockAfter); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos actualizar el stock.")
		return
	}

	var adjustmentID string
	var createdAt time.Time
	if err = tx.QueryRow(r.Context(), `
		INSERT INTO inventory_adjustments(
			organization_id,location_id,inventory_item_id,movement_type,reason,
			quantity,stock_before,stock_after,observation,created_by
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id,created_at`,
		s.OrganizationID, s.LocationID, in.InventoryItemID, in.MovementType, in.Reason,
		in.Quantity, stockBefore, stockAfter, in.Observation, s.UserID).
		Scan(&adjustmentID, &createdAt); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar la trazabilidad del ajuste.")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		INSERT INTO stock_movements(
			organization_id,location_id,product_id,inventory_item_id,
			movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by
		)
		VALUES($1,$2,$3,$4,'inventory_adjustment',$5,$6,'inventory_adjustment',$7,$8,$9)`,
		s.OrganizationID, s.LocationID, productID, in.InventoryItemID,
		delta, stockAfter, adjustmentID, in.Observation, s.UserID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el ajuste.")
		return
	}

	a.audit(r, "inventory.adjusted", "inventory_item", in.InventoryItemID)
	writeJSON(w, 201, map[string]any{
		"id": adjustmentID,
		"inventoryItemId": in.InventoryItemID,
		"name": itemName,
		"unit": unit,
		"movementType": in.MovementType,
		"reason": in.Reason,
		"quantity": in.Quantity,
		"stockBefore": stockBefore,
		"stockAfter": stockAfter,
		"observation": in.Observation,
		"createdAt": createdAt,
		"createdByName": s.Name,
	})
}

func (a *API) listInventoryMovements(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	inventoryItemID := strings.TrimSpace(r.URL.Query().Get("inventoryItemId"))
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	rows, err := a.db.Query(r.Context(), `
		SELECT sm.id,sm.inventory_item_id,sm.product_id,COALESCE(p.name,ii.name),
		       sm.movement_type,ia.movement_type,ia.reason,sm.quantity_delta::text,
		       COALESCE(ia.stock_before,sm.balance_after-sm.quantity_delta)::text,sm.balance_after::text,
		       sm.source_type,sm.source_id,
		       COALESCE(
		         CASE
		           WHEN sm.source_type='order' THEN (
		             SELECT o.code
		             FROM orders o
		             WHERE o.id=sm.source_id AND o.organization_id=sm.organization_id
		           )
		           WHEN sm.source_type='inventory_entry' THEN (
		             SELECT ie.code
		             FROM inventory_entries ie
		             WHERE ie.id=sm.source_id AND ie.organization_id=sm.organization_id
		           )
		           WHEN sm.source_type='inventory_adjustment' THEN 'Ajuste de inventario'
		           WHEN sm.source_type='purchase_receipt' THEN (
		             SELECT pr.code||' · '||po.number
		             FROM purchase_receipts pr
		             JOIN purchase_orders po
		               ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		             WHERE pr.id=sm.source_id AND pr.organization_id=sm.organization_id
		           )
		         END,
		         CASE
		           WHEN sm.source_type='order' THEN 'Pedido histórico'
		           WHEN sm.source_type='inventory_entry' THEN 'Entrada histórica'
		           WHEN sm.source_type='inventory_adjustment' THEN 'Ajuste de inventario'
		           WHEN sm.source_type='purchase_receipt' THEN 'Recepción de compra'
		           ELSE 'Referencia no disponible'
		         END
		       ),
		       sm.note,COALESCE(u.full_name,'Usuario no disponible'),sm.created_at
		FROM stock_movements sm
		JOIN inventory_items ii
		  ON ii.id=sm.inventory_item_id AND ii.organization_id=sm.organization_id
		LEFT JOIN products p
		  ON p.id=sm.product_id AND p.organization_id=sm.organization_id
		LEFT JOIN inventory_adjustments ia
		  ON ia.id=sm.source_id AND ia.organization_id=sm.organization_id
		 AND sm.source_type='inventory_adjustment'
		LEFT JOIN users u ON u.id=sm.created_by
		WHERE sm.organization_id=$1 AND sm.location_id=$2
		  AND ($3='' OR sm.inventory_item_id::text=$3)
		  AND ($4='' OR sm.product_id::text=$4)
		ORDER BY sm.created_at DESC
		LIMIT 100`,
		s.OrganizationID, s.LocationID, inventoryItemID, productID)
	if err != nil {
		fail(w, 503, "kardex_unavailable", "No pudimos cargar el Kárdex.")
		return
	}
	defer rows.Close()
	items := []inventoryMovementView{}
	for rows.Next() {
		var item inventoryMovementView
		if err := rows.Scan(
			&item.ID, &item.InventoryItemID, &item.ProductID, &item.ItemName,
			&item.MovementType, &item.AdjustmentType, &item.Reason, &item.QuantityDelta,
			&item.BalanceBefore, &item.BalanceAfter, &item.SourceType, &item.SourceID,
			&item.SourceReference, &item.Note, &item.CreatedByName, &item.CreatedAt,
		); err != nil {
			fail(w, 503, "kardex_unavailable", "No pudimos cargar el Kárdex.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func preparedQuantityUsage(items []preparedOrderItem) map[string]float64 {
	usage := map[string]float64{}
	for _, item := range items {
		if item.ItemType == "product" && item.ProductID != nil {
			usage[*item.ProductID] += item.Qty
			continue
		}
		if item.ItemType == "combo" {
			for _, selection := range item.Selections {
				usage[selection.ProductID] += item.Qty
			}
		}
	}
	return usage
}

func quantityUsageDelta(previous, next map[string]float64) map[string]float64 {
	ids := map[string]bool{}
	for id := range previous { ids[id] = true }
	for id := range next { ids[id] = true }
	delta := map[string]float64{}
	for id := range ids {
		change := next[id] - previous[id]
		if math.Abs(change) > 0.000001 {
			delta[id] = change
		}
	}
	return delta
}

func loadOrderQuantityUsage(ctx context.Context, tx pgx.Tx, organizationID, orderID string) (map[string]float64, error) {
	usage := map[string]float64{}
	rows, err := tx.Query(ctx, `
		SELECT product_id::text,qty::float8
		FROM order_items
		WHERE order_id=$1 AND organization_id=$2 AND item_type='product' AND product_id IS NOT NULL
		UNION ALL
		SELECT s.option_product_id::text,oi.qty::float8
		FROM order_item_combo_selections s
		JOIN order_items oi ON oi.id=s.order_item_id AND oi.organization_id=s.organization_id
		WHERE oi.order_id=$1 AND oi.organization_id=$2`, orderID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var productID string
		var qty float64
		if err := rows.Scan(&productID, &qty); err != nil {
			return nil, err
		}
		usage[productID] += qty
	}
	return usage, rows.Err()
}

func (a *API) applyOrderQuantityDelta(ctx context.Context, tx pgx.Tx, s scope, orderID string, delta map[string]float64, movementType string) *orderPreparationError {
	if len(delta) == 0 {
		return nil
	}
	var businessDate string
	if err := tx.QueryRow(ctx, `
		SELECT (o.created_at AT TIME ZONE l.timezone)::date::text
		FROM orders o
		JOIN locations l ON l.id=o.location_id AND l.organization_id=o.organization_id
		WHERE o.id=$1 AND o.organization_id=$2 AND o.location_id=$3`,
		orderID, s.OrganizationID, s.LocationID).Scan(&businessDate); err != nil {
		return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos determinar la fecha operativa del pedido."}
	}

	productIDs := make([]string, 0, len(delta))
	for id := range delta { productIDs = append(productIDs, id) }
	sort.Strings(productIDs)

	for _, productID := range productIDs {
		change := delta[productID]
		var name, control string
		if err := tx.QueryRow(ctx, `
			SELECT name,quantity_control
			FROM products
			WHERE id=$1 AND organization_id=$2
			FOR UPDATE`, productID, s.OrganizationID).Scan(&name, &control); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &orderPreparationError{Status: 409, Code: "product_unavailable", Message: "Un producto del pedido ya no existe."}
			}
			return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos validar la cantidad del producto."}
		}

		switch control {
		case "none":
			continue
		case "portions":
			if math.Abs(change-math.Round(change)) > 0.000001 {
				return &orderPreparationError{Status: 400, Code: "invalid_portion_quantity", Message: name + " solo se vende por porciones enteras."}
			}
			var portionQuantity *int
			var soldQuantity int
			var manualStatus string
			err := tx.QueryRow(ctx, `
				SELECT portion_quantity,sold_quantity,manual_status
				FROM product_availability
				WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date
				FOR UPDATE`, s.OrganizationID, s.LocationID, productID, businessDate).Scan(&portionQuantity, &soldQuantity, &manualStatus)
			if errors.Is(err, pgx.ErrNoRows) {
				return &orderPreparationError{Status: 409, Code: "portions_unavailable", Message: name + " no tiene porciones cargadas para este día."}
			}
			if err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos validar las porciones disponibles."}
			}
			step := int(math.Round(change))
			nextSold := soldQuantity + step
			if step > 0 && (manualStatus == "sold_out" || portionQuantity == nil || nextSold > *portionQuantity) {
				return &orderPreparationError{Status: 409, Code: "insufficient_portions", Message: "No quedan suficientes porciones de " + name + "."}
			}
			if nextSold < 0 {
				return &orderPreparationError{Status: 503, Code: "quantity_inconsistent", Message: "No pudimos reconciliar las porciones del pedido."}
			}
			if _, err := tx.Exec(ctx, `
				UPDATE product_availability
				SET sold_quantity=$5,updated_at=now()
				WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date`,
				s.OrganizationID, s.LocationID, productID, businessDate, nextSold); err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos actualizar las porciones disponibles."}
			}
		case "inventory":
			var inventoryItemID string
			err := tx.QueryRow(ctx, `
				SELECT id
				FROM inventory_items
				WHERE organization_id=$1 AND product_id=$2 AND active
				FOR UPDATE`, s.OrganizationID, productID).Scan(&inventoryItemID)
			if errors.Is(err, pgx.ErrNoRows) {
				return &orderPreparationError{Status: 409, Code: "inventory_not_linked", Message: name + " todavía no tiene inventario registrado."}
			}
			if err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos validar el inventario del producto."}
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
				VALUES($1,$2,$3,0)
				ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
				s.OrganizationID, s.LocationID, inventoryItemID); err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos preparar el saldo del producto."}
			}
			var balance float64
			if err := tx.QueryRow(ctx, `
				SELECT quantity::float8
				FROM stock_balances
				WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
				FOR UPDATE`, s.OrganizationID, s.LocationID, inventoryItemID).Scan(&balance); err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos bloquear el saldo del producto."}
			}
			balanceDelta := -change
			nextBalance := balance + balanceDelta
			if nextBalance < -0.000001 {
				return &orderPreparationError{Status: 409, Code: "insufficient_stock", Message: "No hay stock suficiente de " + name + "."}
			}
			if nextBalance < 0 { nextBalance = 0 }
			if _, err := tx.Exec(ctx, `
				UPDATE stock_balances
				SET quantity=$4,updated_at=now()
				WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
				s.OrganizationID, s.LocationID, inventoryItemID, nextBalance); err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos actualizar el stock del producto."}
			}
			if math.Abs(balanceDelta) > 0.000001 {
				if _, err := tx.Exec(ctx, `
					INSERT INTO stock_movements(organization_id,location_id,product_id,inventory_item_id,movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by)
					VALUES($1,$2,$3,$4,$5,$6,$7,'order',$8,$9,$10)`,
					s.OrganizationID, s.LocationID, productID, inventoryItemID, movementType, balanceDelta, nextBalance, orderID, "Pedido "+orderID, s.UserID); err != nil {
					return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos registrar el movimiento de Kárdex."}
				}
			}
		default:
			return &orderPreparationError{Status: 503, Code: "quantity_control_invalid", Message: "El producto tiene un control de cantidad inválido."}
		}
	}
	return nil
}
