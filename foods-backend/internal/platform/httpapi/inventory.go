package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

type inventoryItemView struct {
	ProductID       string  `json:"productId"`
	SKU             string  `json:"sku"`
	Name            string  `json:"name"`
	CategoryName    *string `json:"categoryName"`
	Active          bool    `json:"active"`
	Unit            string  `json:"unit"`
	Quantity        string  `json:"quantity"`
	MinimumStock    string  `json:"minimumStock"`
	Status          string  `json:"status"`
	UpdatedAt       string  `json:"updatedAt"`
	QuantityControl string  `json:"quantityControl"`
}

type inventoryProductOption struct {
	ID              string  `json:"id"`
	SKU             string  `json:"sku"`
	Name            string  `json:"name"`
	CategoryName    *string `json:"categoryName"`
	QuantityControl string  `json:"quantityControl"`
}

type inventoryNewProductInput struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CategoryID  *string `json:"categoryId"`
	Price       string  `json:"price"`
}

type inventoryEntryInput struct {
	ProductID    string                    `json:"productId"`
	NewProduct   *inventoryNewProductInput `json:"newProduct"`
	Quantity     float64                   `json:"quantity"`
	Unit         string                    `json:"unit"`
	MinimumStock float64                   `json:"minimumStock"`
	Note         string                    `json:"note"`
}

type inventoryMovementView struct {
	ID            string `json:"id"`
	ProductID     string `json:"productId"`
	ProductName   string `json:"productName"`
	MovementType  string `json:"movementType"`
	QuantityDelta string `json:"quantityDelta"`
	BalanceAfter  string `json:"balanceAfter"`
	SourceType    string `json:"sourceType"`
	SourceID      string `json:"sourceId"`
	Note          string `json:"note"`
	CreatedAt     string `json:"createdAt"`
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
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.Unit = strings.TrimSpace(in.Unit)
	in.Note = strings.TrimSpace(in.Note)
	if in.Quantity <= 0 {
		return in, "La cantidad de entrada debe ser mayor que cero."
	}
	if in.MinimumStock < 0 {
		return in, "El stock mínimo no puede ser negativo."
	}
	if in.Unit == "" {
		in.Unit = "und"
	}
	hasExisting := in.ProductID != ""
	hasNew := in.NewProduct != nil
	if hasExisting == hasNew {
		return in, "Selecciona un producto existente o crea uno nuevo."
	}
	if in.NewProduct != nil {
		in.NewProduct.Name = strings.TrimSpace(in.NewProduct.Name)
		in.NewProduct.SKU = strings.TrimSpace(in.NewProduct.SKU)
		in.NewProduct.Description = strings.TrimSpace(in.NewProduct.Description)
		in.NewProduct.Price = strings.TrimSpace(in.NewProduct.Price)
		if in.NewProduct.Name == "" || in.NewProduct.Price == "" {
			return in, "El nuevo producto necesita nombre y precio de venta."
		}
	}
	return in, ""
}

func (a *API) listInventory(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	search := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	var total int
	if err := a.db.QueryRow(r.Context(), `
		SELECT count(*)
		FROM products p
		WHERE p.organization_id=$1 AND p.quantity_control='inventory'
		  AND (p.name ILIKE $2 OR p.sku ILIKE $2)`, s.OrganizationID, search).Scan(&total); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos cargar el inventario.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT p.id,p.sku,p.name,c.name,p.active,
		       COALESCE(ii.unit,'und'),COALESCE(sb.quantity,0)::text,COALESCE(ii.minimum_stock,0)::text,
		       CASE
		         WHEN COALESCE(sb.quantity,0)<=0 THEN 'out'
		         WHEN COALESCE(ii.minimum_stock,0)>0 AND COALESCE(sb.quantity,0)<=ii.minimum_stock THEN 'low'
		         ELSE 'ok'
		       END,
		       to_char(COALESCE(sb.updated_at,p.updated_at),'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       p.quantity_control
		FROM products p
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		LEFT JOIN inventory_items ii ON ii.organization_id=p.organization_id AND ii.product_id=p.id
		LEFT JOIN stock_balances sb ON sb.organization_id=p.organization_id AND sb.location_id=$2 AND sb.inventory_item_id=ii.id
		WHERE p.organization_id=$1 AND p.quantity_control='inventory'
		  AND (p.name ILIKE $3 OR p.sku ILIKE $3)
		ORDER BY p.active DESC,p.name
		LIMIT $4 OFFSET $5`, s.OrganizationID, s.LocationID, search, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos cargar el inventario.")
		return
	}
	defer rows.Close()
	items := []inventoryItemView{}
	for rows.Next() {
		var item inventoryItemView
		if err := rows.Scan(&item.ProductID, &item.SKU, &item.Name, &item.CategoryName, &item.Active, &item.Unit, &item.Quantity, &item.MinimumStock, &item.Status, &item.UpdatedAt, &item.QuantityControl); err != nil {
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
		SELECT p.id,p.sku,p.name,c.name,p.quantity_control
		FROM products p
		LEFT JOIN menu_categories c ON c.id=p.category_id AND c.organization_id=p.organization_id
		WHERE p.organization_id=$1 AND p.active
		  AND p.quantity_control IN ('none','inventory')
		  AND NOT EXISTS (SELECT 1 FROM menu_combos mc WHERE mc.product_id=p.id AND mc.organization_id=p.organization_id)
		  AND (p.name ILIKE $2 OR p.sku ILIKE $2)
		ORDER BY p.name
		LIMIT 100`, s.OrganizationID, search)
	if err != nil {
		fail(w, 503, "inventory_products_unavailable", "No pudimos cargar los productos.")
		return
	}
	defer rows.Close()
	items := []inventoryProductOption{}
	for rows.Next() {
		var item inventoryProductOption
		if err := rows.Scan(&item.ID, &item.SKU, &item.Name, &item.CategoryName, &item.QuantityControl); err != nil {
			fail(w, 503, "inventory_products_unavailable", "No pudimos cargar los productos.")
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

	productID := in.ProductID
	var productSKU, productName string
	createdProduct := false
	if in.NewProduct != nil {
		productIn := productInput{
			SKU:             in.NewProduct.SKU,
			Name:            in.NewProduct.Name,
			Description:     in.NewProduct.Description,
			CategoryID:      in.NewProduct.CategoryID,
			Price:           in.NewProduct.Price,
			QuantityControl: "inventory",
		}
		if _, invalidProduct := normalizeProduct(productIn); invalidProduct != "" {
			fail(w, 400, "invalid_product", invalidProduct)
			return
		}
		err = tx.QueryRow(r.Context(), `
			INSERT INTO products(organization_id,category_id,sku,name,description,price,active,quantity_control)
			VALUES($1,$2,COALESCE(NULLIF($3,''),'PROD-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,8))),$4,$5,$6,true,'inventory')
			RETURNING id,sku,name`,
			s.OrganizationID, in.NewProduct.CategoryID, in.NewProduct.SKU, in.NewProduct.Name, in.NewProduct.Description, in.NewProduct.Price,
		).Scan(&productID, &productSKU, &productName)
		if err != nil {
			fail(w, 409, "product_conflict", "No pudimos crear el producto. Revisa categoría, código y precio.")
			return
		}
		createdProduct = true
	} else {
		var quantityControl string
		err = tx.QueryRow(r.Context(), `
			SELECT id,sku,name,quantity_control
			FROM products
			WHERE id=$1 AND organization_id=$2 AND active
			FOR UPDATE`, productID, s.OrganizationID).Scan(&productID, &productSKU, &productName, &quantityControl)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "product_not_found", "El producto no existe o está inactivo.")
			return
		}
		if err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos validar el producto.")
			return
		}
		if quantityControl == "portions" {
			fail(w, 409, "quantity_control_conflict", "Ese producto se controla por porciones y no puede recibir entradas de inventario.")
			return
		}
		if quantityControl == "none" {
			usedByOpenOrder, usageErr := productHasCancellableOrderUsage(r.Context(), tx, s.OrganizationID, productID)
			if usageErr != nil {
				fail(w, 503, "inventory_unavailable", "No pudimos validar los pedidos abiertos del producto.")
				return
			}
			if usedByOpenOrder {
				fail(w, 409, "quantity_control_open_orders", "Cierra o cancela los pedidos abiertos de este producto antes de activar Inventario.")
				return
			}
			if _, err = tx.Exec(r.Context(), `
				UPDATE products SET quantity_control='inventory',updated_at=now()
				WHERE id=$1 AND organization_id=$2`, productID, s.OrganizationID); err != nil {
				fail(w, 503, "inventory_unavailable", "No pudimos activar el control de inventario.")
				return
			}
		}
	}

	var inventoryItemID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO inventory_items(organization_id,sku,name,unit,minimum_stock,active,product_id)
		VALUES($1,'SELL-'||upper(substr(replace($2::text,'-',''),1,12)),$3,$4,$5,true,$2)
		ON CONFLICT (organization_id,product_id) WHERE product_id IS NOT NULL
		DO UPDATE SET name=EXCLUDED.name,unit=EXCLUDED.unit,minimum_stock=EXCLUDED.minimum_stock,active=true
		RETURNING id`,
		s.OrganizationID, productID, productName, in.Unit, in.MinimumStock).Scan(&inventoryItemID)
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos vincular el producto al inventario.")
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
	balanceAfter := current + in.Quantity
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
		INSERT INTO inventory_entries(organization_id,location_id,product_id,inventory_item_id,quantity,unit,note,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id`,
		s.OrganizationID, s.LocationID, productID, inventoryItemID, in.Quantity, in.Unit, in.Note, s.UserID).Scan(&entryID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el documento de entrada.")
		return
	}

	if _, err = tx.Exec(r.Context(), `
		INSERT INTO stock_movements(organization_id,location_id,product_id,inventory_item_id,movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by)
		VALUES($1,$2,$3,$4,'entry',$5,$6,'inventory_entry',$7,$8,$9)`,
		s.OrganizationID, s.LocationID, productID, inventoryItemID, in.Quantity, balanceAfter, entryID, in.Note, s.UserID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos registrar la entrada.")
		return
	}
	if createdProduct {
		a.audit(r, "product.created_from_inventory", "product", productID)
	}
	a.audit(r, "inventory.entry_created", "product", productID)
	writeJSON(w, 201, map[string]any{
		"id": entryID, "productId": productID, "sku": productSKU, "name": productName,
		"quantity": in.Quantity, "unit": in.Unit, "balance": balanceAfter, "createdProduct": createdProduct,
	})
}

func (a *API) listInventoryMovements(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	rows, err := a.db.Query(r.Context(), `
		SELECT sm.id,sm.product_id,p.name,sm.movement_type,sm.quantity_delta::text,sm.balance_after::text,
		       sm.source_type,sm.source_id,sm.note,to_char(sm.created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		FROM stock_movements sm
		JOIN products p ON p.id=sm.product_id AND p.organization_id=sm.organization_id
		WHERE sm.organization_id=$1 AND sm.location_id=$2
		  AND ($3='' OR sm.product_id::text=$3)
		ORDER BY sm.created_at DESC
		LIMIT 100`, s.OrganizationID, s.LocationID, productID)
	if err != nil {
		fail(w, 503, "kardex_unavailable", "No pudimos cargar el Kárdex.")
		return
	}
	defer rows.Close()
	items := []inventoryMovementView{}
	for rows.Next() {
		var item inventoryMovementView
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.MovementType, &item.QuantityDelta, &item.BalanceAfter, &item.SourceType, &item.SourceID, &item.Note, &item.CreatedAt); err != nil {
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
