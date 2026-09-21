package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type supplierView struct {
	ID     string `json:"id"`
	TaxID  string `json:"taxId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Active bool   `json:"active"`
}

type supplierInput struct {
	TaxID string `json:"taxId"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type purchaseOrderItemInput struct {
	InventoryItemID string  `json:"inventoryItemId"`
	PresentationID  string  `json:"presentationId"`
	Quantity        float64 `json:"quantity"`
	UnitCost        string  `json:"unitCost"`
}

type purchaseOrderInput struct {
	SupplierID string                   `json:"supplierId"`
	ExpectedAt string                   `json:"expectedAt"`
	Notes      string                   `json:"notes"`
	Items      []purchaseOrderItemInput `json:"items"`
}

type purchaseReceiptLineInput struct {
	PurchaseOrderItemID string  `json:"purchaseOrderItemId"`
	Quantity            float64 `json:"quantity"`
}

type purchaseReceiptInput struct {
	Notes string                     `json:"notes"`
	IdempotencyKey string            `json:"idempotencyKey"`
	Items []purchaseReceiptLineInput `json:"items"`
}

type purchaseInventoryItemInput struct {
	NewProduct           *inventoryNewProductInput    `json:"newProduct"`
	NewIngredient        *inventoryNewIngredientInput `json:"newIngredient"`
	Unit                 string                       `json:"unit"`
	PresentationType     string                       `json:"presentationType"`
	UnitsPerPresentation float64                      `json:"unitsPerPresentation"`
	MinimumStock         float64                      `json:"minimumStock"`
}

type purchaseOrderSummary struct {
	ID           string     `json:"id"`
	Number       string     `json:"number"`
	SupplierID   string     `json:"supplierId"`
	SupplierName string     `json:"supplierName"`
	Status       string     `json:"status"`
	Total        string     `json:"total"`
	Notes        string     `json:"notes"`
	ExpectedAt   *string    `json:"expectedAt"`
	ItemCount    int        `json:"itemCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	ApprovedAt   *time.Time `json:"approvedAt"`
	ReceivedAt   *time.Time `json:"receivedAt"`
	CancelledAt  *time.Time `json:"cancelledAt"`
}

type purchaseOrderItemView struct {
	ID                   string `json:"id"`
	InventoryItemID      string `json:"inventoryItemId"`
	ItemName             string `json:"itemName"`
	SKU                  string `json:"sku"`
	Unit                 string `json:"unit"`
	PresentationID       string `json:"presentationId"`
	PresentationType     string `json:"presentationType"`
	UnitsPerPresentation string `json:"unitsPerPresentation"`
	Quantity             string `json:"quantity"`
	ReceivedQuantity     string `json:"receivedQuantity"`
	PendingQuantity      string `json:"pendingQuantity"`
	StockQuantity        string `json:"stockQuantity"`
	UnitCost             string `json:"unitCost"`
	LineTotal            string `json:"lineTotal"`
}

type purchaseOrderDetail struct {
	purchaseOrderSummary
	Items []purchaseOrderItemView `json:"items"`
}

var purchaseMoneyPattern = regexp.MustCompile(`^[0-9]+([.][0-9]{1,4})?$`)

func normalizeSupplier(in supplierInput) (supplierInput, string) {
	in.Name = strings.TrimSpace(in.Name)
	in.TaxID = strings.TrimSpace(in.TaxID)
	in.Email = strings.TrimSpace(in.Email)
	in.Phone = strings.TrimSpace(in.Phone)
	if in.Name == "" {
		return in, "El nombre del proveedor es obligatorio."
	}
	if len(in.Name) > 180 || len(in.TaxID) > 11 || len(in.Email) > 180 || len(in.Phone) > 40 {
		return in, "Revisa la longitud de los datos del proveedor."
	}
	if in.Email != "" && !strings.Contains(in.Email, "@") {
		return in, "El correo del proveedor no es válido."
	}
	return in, ""
}

func normalizePurchaseOrder(in purchaseOrderInput) (purchaseOrderInput, string) {
	in.SupplierID = strings.TrimSpace(in.SupplierID)
	in.ExpectedAt = strings.TrimSpace(in.ExpectedAt)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.SupplierID == "" {
		return in, "Selecciona un proveedor."
	}
	if in.ExpectedAt != "" {
		if _, err := time.Parse("2006-01-02", in.ExpectedAt); err != nil {
			return in, "La fecha esperada no es válida."
		}
	}
	if len(in.Notes) > 500 {
		return in, "Las notas no pueden superar 500 caracteres."
	}
	if len(in.Items) == 0 {
		return in, "Agrega al menos un artículo a la orden."
	}
	if len(in.Items) > 100 {
		return in, "Una orden no puede superar 100 líneas."
	}
	seen := map[string]bool{}
	for index := range in.Items {
		item := &in.Items[index]
		item.InventoryItemID = strings.TrimSpace(item.InventoryItemID)
		item.PresentationID = strings.TrimSpace(item.PresentationID)
		item.UnitCost = strings.TrimSpace(item.UnitCost)
		item.Quantity = math.Round(item.Quantity*1000) / 1000
		if item.InventoryItemID == "" || item.PresentationID == "" || item.Quantity <= 0 {
			return in, "Cada línea necesita artículo, presentación y cantidad mayor que cero."
		}
		if !purchaseMoneyPattern.MatchString(item.UnitCost) {
			return in, "Cada línea necesita un costo unitario válido."
		}
		key := item.InventoryItemID + ":" + item.PresentationID
		if seen[key] {
			return in, "No repitas el mismo artículo con la misma presentación."
		}
		seen[key] = true
	}
	return in, ""
}

func (a *API) listSuppliers(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	where := `organization_id=$1
		AND ($2='' OR name ILIKE '%'||$2||'%' OR COALESCE(tax_id,'') ILIKE '%'||$2||'%')
		AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM suppliers WHERE `+where, s.OrganizationID, q, status).Scan(&total); err != nil {
		fail(w, 503, "suppliers_unavailable", "No pudimos cargar los proveedores.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT id,COALESCE(tax_id,''),name,COALESCE(email,''),COALESCE(phone,''),active
		FROM suppliers
		WHERE `+where+`
		ORDER BY active DESC,name
		LIMIT $4 OFFSET $5`, s.OrganizationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "suppliers_unavailable", "No pudimos cargar los proveedores.")
		return
	}
	defer rows.Close()
	items := []supplierView{}
	for rows.Next() {
		var item supplierView
		if err := rows.Scan(&item.ID, &item.TaxID, &item.Name, &item.Email, &item.Phone, &item.Active); err != nil {
			fail(w, 503, "suppliers_unavailable", "No pudimos cargar los proveedores.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) createSupplier(w http.ResponseWriter, r *http.Request) {
	a.saveSupplier(w, r, false)
}

func (a *API) updateSupplier(w http.ResponseWriter, r *http.Request) {
	a.saveSupplier(w, r, true)
}

func (a *API) saveSupplier(w http.ResponseWriter, r *http.Request, updating bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in supplierInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_supplier", "Revisa los datos del proveedor.")
		return
	}
	var invalid string
	in, invalid = normalizeSupplier(in)
	if invalid != "" {
		fail(w, 400, "invalid_supplier", invalid)
		return
	}
	var item supplierView
	var err error
	if updating {
		err = a.db.QueryRow(r.Context(), `
			UPDATE suppliers
			SET tax_id=NULLIF($3,''),name=$4,email=NULLIF($5,''),phone=NULLIF($6,'')
			WHERE id=$1 AND organization_id=$2
			RETURNING id,COALESCE(tax_id,''),name,COALESCE(email,''),COALESCE(phone,''),active`,
			r.PathValue("id"), s.OrganizationID, in.TaxID, in.Name, in.Email, in.Phone,
		).Scan(&item.ID, &item.TaxID, &item.Name, &item.Email, &item.Phone, &item.Active)
	} else {
		err = a.db.QueryRow(r.Context(), `
			INSERT INTO suppliers(organization_id,tax_id,name,email,phone)
			VALUES($1,NULLIF($2,''),$3,NULLIF($4,''),NULLIF($5,''))
			RETURNING id,COALESCE(tax_id,''),name,COALESCE(email,''),COALESCE(phone,''),active`,
			s.OrganizationID, in.TaxID, in.Name, in.Email, in.Phone,
		).Scan(&item.ID, &item.TaxID, &item.Name, &item.Email, &item.Phone, &item.Active)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "supplier_not_found", "El proveedor no existe.")
		return
	}
	if err != nil {
		fail(w, 409, "supplier_conflict", "No pudimos guardar el proveedor. Revisa el RUC y los datos ingresados.")
		return
	}
	a.audit(r, map[bool]string{true: "supplier.updated", false: "supplier.created"}[updating], "supplier", item.ID)
	writeJSON(w, map[bool]int{true: 200, false: 201}[updating], item)
}

func (a *API) updateSupplierStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_status", "El estado no es válido.")
		return
	}
	if !in.Active {
		var open bool
		if err := a.db.QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM purchase_orders
				WHERE organization_id=$1 AND supplier_id=$2
				  AND status IN ('draft','pending_approval','approved','partially_received')
			)`, s.OrganizationID, r.PathValue("id")).Scan(&open); err != nil {
			fail(w, 503, "supplier_unavailable", "No pudimos validar el proveedor.")
			return
		}
		if open {
			fail(w, 409, "supplier_in_use", "El proveedor tiene órdenes abiertas y no puede desactivarse.")
			return
		}
	}
	tag, err := a.db.Exec(r.Context(), `
		UPDATE suppliers SET active=$3
		WHERE id=$1 AND organization_id=$2`,
		r.PathValue("id"), s.OrganizationID, in.Active)
	if err != nil || tag.RowsAffected() == 0 {
		fail(w, 404, "supplier_not_found", "El proveedor no existe.")
		return
	}
	a.audit(r, "supplier.status_updated", "supplier", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createPurchaseInventoryItem(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var input purchaseInventoryItemInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		fail(w, 400, "invalid_purchase_item", "Revisa los datos del artículo.")
		return
	}
	normalized, invalid := normalizeInventoryEntry(inventoryEntryInput{
		NewProduct: input.NewProduct,
		NewIngredient: input.NewIngredient,
		Quantity: 1,
		Unit: input.Unit,
		PresentationType: input.PresentationType,
		UnitsPerPresentation: input.UnitsPerPresentation,
		MinimumStock: input.MinimumStock,
	})
	if invalid != "" {
		fail(w, 400, "invalid_purchase_item", invalid)
		return
	}
	if normalized.NewProduct == nil && normalized.NewIngredient == nil {
		fail(w, 400, "invalid_purchase_item", "Selecciona Nuevo producto vendible o Nuevo insumo.")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos crear el artículo.")
		return
	}
	defer tx.Rollback(r.Context())

	created, createErr := createInventoryCatalogItem(r.Context(), tx, s.OrganizationID, normalized)
	if createErr != nil {
		fail(w, createErr.Status, createErr.Code, createErr.Message)
		return
	}
	if _, err = ensureInventoryPresentation(
		r.Context(), tx, s.OrganizationID, created.InventoryItemID,
		normalized.PresentationType, normalized.UnitsPerPresentation,
	); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos guardar la presentación del artículo.")
		return
	}
	if _, err = tx.Exec(r.Context(), `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
		VALUES($1,$2,$3,0)
		ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
		s.OrganizationID, s.LocationID, created.InventoryItemID); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos preparar el artículo con stock cero.")
		return
	}

	rows, err := tx.Query(r.Context(), `
		SELECT id,presentation_type,units_per_presentation::text
		FROM inventory_presentations
		WHERE organization_id=$1 AND inventory_item_id=$2 AND active
		ORDER BY CASE presentation_type WHEN 'unit' THEN 0 WHEN 'package' THEN 1 ELSE 2 END,units_per_presentation`,
		s.OrganizationID, created.InventoryItemID)
	if err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos cargar las presentaciones del artículo.")
		return
	}
	presentations := []inventoryPresentationOption{}
	for rows.Next() {
		var p inventoryPresentationOption
		if err := rows.Scan(&p.ID, &p.PresentationType, &p.UnitsPerPresentation); err != nil {
			rows.Close()
			fail(w, 503, "inventory_unavailable", "No pudimos cargar las presentaciones del artículo.")
			return
		}
		presentations = append(presentations, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(w, 503, "inventory_unavailable", "No pudimos cargar las presentaciones del artículo.")
		return
	}
	rows.Close()

	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "inventory_unavailable", "No pudimos crear el artículo.")
		return
	}
	if created.ProductID != nil {
		a.audit(r, "product.created_from_purchase", "product", *created.ProductID)
	} else {
		a.audit(r, "inventory.ingredient_created_from_purchase", "inventory_item", created.InventoryItemID)
	}
	var quantityControl *string
	if created.ProductID != nil {
		control := "inventory"
		quantityControl = &control
	}
	minimumStock := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", normalized.MinimumStock), "0"), ".")
	if minimumStock == "" {
		minimumStock = "0"
	}
	writeJSON(w, 201, inventoryProductOption{
		ID: created.InventoryItemID,
		ProductID: created.ProductID,
		SKU: created.SKU,
		Name: created.Name,
		Kind: created.Kind,
		CategoryName: nil,
		QuantityControl: quantityControl,
		Unit: created.Unit,
		Quantity: "0",
		MinimumStock: minimumStock,
		Presentations: presentations,
	})
}

func (a *API) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	where := `po.organization_id=$1 AND po.location_id=$2
		AND ($3='' OR po.number ILIKE '%'||$3||'%' OR sp.name ILIKE '%'||$3||'%')
		AND (
		  $4=''
		  OR ($4='receivable' AND po.status IN ('approved','partially_received'))
		  OR po.status=$4
		)`
	var total int
	if err := a.db.QueryRow(r.Context(), `
		SELECT count(*)
		FROM purchase_orders po
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE `+where, s.OrganizationID, s.LocationID, q, status).Scan(&total); err != nil {
		fail(w, 503, "purchases_unavailable", "No pudimos cargar las órdenes de compra.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT po.id,po.number,po.supplier_id,sp.name,po.status,po.total::text,po.notes,
		       CASE WHEN po.expected_at IS NULL THEN NULL ELSE po.expected_at::text END,
		       (SELECT count(*) FROM purchase_order_items poi WHERE poi.organization_id=po.organization_id AND poi.purchase_order_id=po.id),
		       po.created_at,po.updated_at,po.approved_at,po.received_at,po.cancelled_at
		FROM purchase_orders po
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE `+where+`
		ORDER BY po.created_at DESC
		LIMIT $5 OFFSET $6`, s.OrganizationID, s.LocationID, q, status, size, (page-1)*size)
	if err != nil {
		fail(w, 503, "purchases_unavailable", "No pudimos cargar las órdenes de compra.")
		return
	}
	defer rows.Close()
	items := []purchaseOrderSummary{}
	for rows.Next() {
		var item purchaseOrderSummary
		if err := rows.Scan(
			&item.ID, &item.Number, &item.SupplierID, &item.SupplierName, &item.Status,
			&item.Total, &item.Notes, &item.ExpectedAt, &item.ItemCount, &item.CreatedAt,
			&item.UpdatedAt, &item.ApprovedAt, &item.ReceivedAt, &item.CancelledAt,
		); err != nil {
			fail(w, 503, "purchases_unavailable", "No pudimos cargar las órdenes de compra.")
			return
		}
		items = append(items, item)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "pageSize": size})
}

func (a *API) getPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var item purchaseOrderDetail
	err := a.db.QueryRow(r.Context(), `
		SELECT po.id,po.number,po.supplier_id,sp.name,po.status,po.total::text,po.notes,
		       CASE WHEN po.expected_at IS NULL THEN NULL ELSE po.expected_at::text END,
		       (SELECT count(*) FROM purchase_order_items poi WHERE poi.organization_id=po.organization_id AND poi.purchase_order_id=po.id),
		       po.created_at,po.updated_at,po.approved_at,po.received_at,po.cancelled_at
		FROM purchase_orders po
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE po.id=$1 AND po.organization_id=$2 AND po.location_id=$3`,
		r.PathValue("id"), s.OrganizationID, s.LocationID,
	).Scan(
		&item.ID, &item.Number, &item.SupplierID, &item.SupplierName, &item.Status,
		&item.Total, &item.Notes, &item.ExpectedAt, &item.ItemCount, &item.CreatedAt,
		&item.UpdatedAt, &item.ApprovedAt, &item.ReceivedAt, &item.CancelledAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "purchase_not_found", "La orden de compra no existe.")
		return
	}
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos cargar la orden de compra.")
		return
	}
	rows, err := a.db.Query(r.Context(), `
		SELECT poi.id,poi.inventory_item_id,COALESCE(p.name,ii.name),COALESCE(p.sku,ii.sku),ii.unit,
		       poi.presentation_id,poi.presentation_type,poi.units_per_presentation::text,
		       poi.quantity::text,poi.received_quantity::text,(poi.quantity-poi.received_quantity)::text,
		       poi.stock_quantity::text,poi.unit_cost::text,poi.line_total::text
		FROM purchase_order_items poi
		JOIN inventory_items ii ON ii.id=poi.inventory_item_id AND ii.organization_id=poi.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE poi.purchase_order_id=$1 AND poi.organization_id=$2
		ORDER BY poi.created_at,COALESCE(p.name,ii.name)`,
		item.ID, s.OrganizationID)
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos cargar el detalle de la orden.")
		return
	}
	defer rows.Close()
	item.Items = []purchaseOrderItemView{}
	for rows.Next() {
		var line purchaseOrderItemView
		if err := rows.Scan(
			&line.ID, &line.InventoryItemID, &line.ItemName, &line.SKU, &line.Unit,
			&line.PresentationID, &line.PresentationType, &line.UnitsPerPresentation,
			&line.Quantity, &line.ReceivedQuantity, &line.PendingQuantity,
			&line.StockQuantity, &line.UnitCost, &line.LineTotal,
		); err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos cargar el detalle de la orden.")
			return
		}
		item.Items = append(item.Items, line)
	}
	writeJSON(w, 200, item)
}

func (a *API) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.savePurchaseOrder(w, r, false)
}

func (a *API) updatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	a.savePurchaseOrder(w, r, true)
}

func (a *API) savePurchaseOrder(w http.ResponseWriter, r *http.Request, updating bool) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in purchaseOrderInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_purchase", "Revisa los datos de la orden.")
		return
	}
	var invalid string
	in, invalid = normalizePurchaseOrder(in)
	if invalid != "" {
		fail(w, 400, "invalid_purchase", invalid)
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos guardar la orden.")
		return
	}
	defer tx.Rollback(r.Context())

	var supplierActive bool
	if err = tx.QueryRow(r.Context(), `
		SELECT active FROM suppliers WHERE id=$1 AND organization_id=$2`,
		in.SupplierID, s.OrganizationID).Scan(&supplierActive); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 400, "supplier_not_found", "El proveedor seleccionado no existe.")
		return
	} else if err != nil {
		fail(w, 503, "supplier_unavailable", "No pudimos validar el proveedor.")
		return
	} else if !supplierActive {
		fail(w, 409, "supplier_inactive", "El proveedor seleccionado está inactivo.")
		return
	}

	var id, number string
	if updating {
		var status string
		err = tx.QueryRow(r.Context(), `
			SELECT number,status
			FROM purchase_orders
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
			FOR UPDATE`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&number, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 404, "purchase_not_found", "La orden de compra no existe.")
			return
		}
		if err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos validar la orden.")
			return
		}
		if status != "draft" {
			fail(w, 409, "purchase_locked", "Solo una orden en borrador puede editarse.")
			return
		}
		id = r.PathValue("id")
		if _, err = tx.Exec(r.Context(), `
			UPDATE purchase_orders
			SET supplier_id=$4,expected_at=NULLIF($5,'')::date,notes=$6,total=0,updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
			id, s.OrganizationID, s.LocationID, in.SupplierID, in.ExpectedAt, in.Notes); err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos actualizar la orden.")
			return
		}
		if _, err = tx.Exec(r.Context(), `
			DELETE FROM purchase_order_items
			WHERE purchase_order_id=$1 AND organization_id=$2`, id, s.OrganizationID); err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos actualizar las líneas de la orden.")
			return
		}
	} else {
		err = tx.QueryRow(r.Context(), `
			INSERT INTO purchase_orders(
				organization_id,location_id,supplier_id,number,status,total,notes,expected_at
			)
			VALUES(
				$1,$2,$3,
				'OC-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,8)),
				'draft',0,$4,NULLIF($5,'')::date
			)
			RETURNING id,number`,
			s.OrganizationID, s.LocationID, in.SupplierID, in.Notes, in.ExpectedAt,
		).Scan(&id, &number)
		if err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos crear la orden.")
			return
		}
	}

	for _, line := range in.Items {
		var presentationType string
		var unitsPerPresentation float64
		err = tx.QueryRow(r.Context(), `
			SELECT ip.presentation_type,ip.units_per_presentation::float8
			FROM inventory_presentations ip
			JOIN inventory_items ii
			  ON ii.id=ip.inventory_item_id AND ii.organization_id=ip.organization_id
			WHERE ip.id=$1 AND ip.inventory_item_id=$2 AND ip.organization_id=$3
			  AND ip.active AND ii.active`,
			line.PresentationID, line.InventoryItemID, s.OrganizationID,
		).Scan(&presentationType, &unitsPerPresentation)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 409, "purchase_item_invalid", "Un artículo o presentación de la orden ya no está disponible.")
			return
		}
		if err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos validar los artículos de la orden.")
			return
		}
		if presentationType != "unit" && math.Abs(line.Quantity-math.Round(line.Quantity)) > 0.000001 {
			fail(w, 400, "invalid_purchase_quantity", "Paquetes y cajas deben comprarse en cantidades enteras.")
			return
		}
		if _, err = tx.Exec(r.Context(), `
			INSERT INTO purchase_order_items(
				organization_id,purchase_order_id,inventory_item_id,presentation_id,
				quantity,presentation_type,units_per_presentation,unit_cost
			)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8::numeric)`,
			s.OrganizationID, id, line.InventoryItemID, line.PresentationID,
			line.Quantity, presentationType, unitsPerPresentation, line.UnitCost,
		); err != nil {
			fail(w, 409, "purchase_item_conflict", "No pudimos agregar una de las líneas de la orden.")
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE purchase_orders
		SET total=COALESCE((
			SELECT sum(line_total)
			FROM purchase_order_items
			WHERE purchase_order_id=$1 AND organization_id=$2
		),0),updated_at=now()
		WHERE id=$1 AND organization_id=$2`, id, s.OrganizationID); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos calcular el total de la orden.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos guardar la orden.")
		return
	}
	a.audit(r, map[bool]string{true: "purchase.updated", false: "purchase.created"}[updating], "purchase_order", id)
	writeJSON(w, map[bool]int{true: 200, false: 201}[updating], map[string]string{"id": id, "number": number, "status": "draft"})
}

func (a *API) updatePurchaseOrderStatus(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_status", "El estado solicitado no es válido.")
		return
	}
	in.Status = strings.TrimSpace(in.Status)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos actualizar la orden.")
		return
	}
	defer tx.Rollback(r.Context())
	var current string
	if err = tx.QueryRow(r.Context(), `
		SELECT status FROM purchase_orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "purchase_not_found", "La orden de compra no existe.")
		return
	} else if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos validar la orden.")
		return
	}
	allowed := map[string]map[string]bool{
		"draft":            {"pending_approval": true, "cancelled": true},
		"pending_approval": {"draft": true, "cancelled": true},
		"approved":         {"cancelled": true},
	}
	if !allowed[current][in.Status] {
		fail(w, 409, "purchase_transition_invalid", "La orden no puede pasar al estado solicitado.")
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE purchase_orders
		SET status=$4,updated_at=now(),
		    approved_at=CASE WHEN $4='approved' THEN now() WHEN $4='draft' THEN NULL ELSE approved_at END,
		    cancelled_at=CASE WHEN $4='cancelled' THEN now() ELSE cancelled_at END
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		r.PathValue("id"), s.OrganizationID, s.LocationID, in.Status)
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos actualizar el estado de la orden.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos actualizar el estado de la orden.")
		return
	}
	a.audit(r, "purchase.status_updated", "purchase_order", r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) receivePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	var in purchaseReceiptInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		fail(w, 400, "invalid_purchase_receipt", "Revisa las cantidades recibidas.")
		return
	}
	in.Notes = strings.TrimSpace(in.Notes)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if len(in.IdempotencyKey)>120 {
		fail(w,400,"invalid_purchase_receipt","La clave de idempotencia es demasiado larga.")
		return
	}
	if len(in.Notes) > 500 {
		fail(w, 400, "invalid_purchase_receipt", "Las notas no pueden superar 500 caracteres.")
		return
	}
	if len(in.Items) == 0 {
		fail(w, 400, "invalid_purchase_receipt", "Registra al menos una cantidad recibida.")
		return
	}
	if len(in.Items) > 100 {
		fail(w, 400, "invalid_purchase_receipt", "La recepción no puede superar 100 líneas.")
		return
	}
	seen := map[string]bool{}
	for index := range in.Items {
		line := &in.Items[index]
		line.PurchaseOrderItemID = strings.TrimSpace(line.PurchaseOrderItemID)
		line.Quantity = math.Round(line.Quantity*1000) / 1000
		if line.PurchaseOrderItemID == "" || line.Quantity <= 0 {
			fail(w, 400, "invalid_purchase_receipt", "Cada línea recibida necesita artículo y cantidad mayor que cero.")
			return
		}
		if seen[line.PurchaseOrderItemID] {
			fail(w, 400, "invalid_purchase_receipt", "No repitas el mismo artículo en una recepción.")
			return
		}
		seen[line.PurchaseOrderItemID] = true
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos recibir la orden.")
		return
	}
	defer tx.Rollback(r.Context())

	var number, status string
	if err = tx.QueryRow(r.Context(), `
		SELECT number,status
		FROM purchase_orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE`, r.PathValue("id"), s.OrganizationID, s.LocationID).Scan(&number, &status); errors.Is(err, pgx.ErrNoRows) {
		fail(w, 404, "purchase_not_found", "La orden de compra no existe.")
		return
	} else if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos validar la orden.")
		return
	}
	if in.IdempotencyKey!=""{
		var existingID,existingCode string
		var existingStatus string
		err=tx.QueryRow(r.Context(),`
			SELECT pr.id,pr.code,po.status
			FROM purchase_receipts pr
			JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id AND po.location_id=pr.location_id
			WHERE pr.organization_id=$1 AND pr.location_id=$2 AND pr.purchase_order_id=$3 AND pr.idempotency_key=$4
		`,s.OrganizationID,s.LocationID,r.PathValue("id"),in.IdempotencyKey).Scan(&existingID,&existingCode,&existingStatus)
		if err==nil{
			writeJSON(w,200,map[string]any{"id":existingID,"code":existingCode,"purchaseOrderId":r.PathValue("id"),"number":number,"status":existingStatus,"idempotent":true})
			return
		}
		if err!=nil&&!errors.Is(err,pgx.ErrNoRows){fail(w,503,"purchase_unavailable","No pudimos validar la recepción previa.");return}
	}
	if status != "approved" && status != "partially_received" {
		fail(w, 409, "purchase_not_receivable", "Solo una orden aprobada o parcialmente recibida puede recibir mercadería.")
		return
	}

	var receiptID, receiptCode string
	if err = tx.QueryRow(r.Context(), `
		INSERT INTO purchase_receipts(
			organization_id,location_id,purchase_order_id,notes,created_by
		)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id,code`,
		s.OrganizationID, s.LocationID, r.PathValue("id"), in.Notes, s.UserID,
	).Scan(&receiptID, &receiptCode); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos crear la recepción.")
		return
	}

	type receivingLine struct {
		ID                   string
		InventoryItemID      string
		ProductID            *string
		ItemName             string
		PresentationID       string
		PresentationType     string
		UnitsPerPresentation float64
		OrderedQuantity      float64
		ReceivedQuantity     float64
		ReceiveQuantity      float64
		UnitCost             float64
	}
	lines := make([]receivingLine, 0, len(in.Items))
	for _, requested := range in.Items {
		var line receivingLine
		err = tx.QueryRow(r.Context(), `
			SELECT poi.id,poi.inventory_item_id,ii.product_id,COALESCE(p.name,ii.name),
			       poi.presentation_id,poi.presentation_type,poi.units_per_presentation::float8,
			       poi.quantity::float8,poi.received_quantity::float8,poi.unit_cost::float8
			FROM purchase_order_items poi
			JOIN inventory_items ii ON ii.id=poi.inventory_item_id AND ii.organization_id=poi.organization_id
			LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
			WHERE poi.id=$1 AND poi.purchase_order_id=$2 AND poi.organization_id=$3
			FOR UPDATE OF poi`,
			requested.PurchaseOrderItemID, r.PathValue("id"), s.OrganizationID,
		).Scan(
			&line.ID, &line.InventoryItemID, &line.ProductID, &line.ItemName,
			&line.PresentationID, &line.PresentationType, &line.UnitsPerPresentation,
			&line.OrderedQuantity, &line.ReceivedQuantity, &line.UnitCost,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 409, "purchase_receipt_item_invalid", "Una línea ya no pertenece a esta orden.")
			return
		}
		if err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos validar los artículos de la recepción.")
			return
		}
		line.ReceiveQuantity = requested.Quantity
		pending := math.Round((line.OrderedQuantity-line.ReceivedQuantity)*1000) / 1000
		if line.ReceiveQuantity-pending > 0.000001 {
			fail(w, 409, "purchase_receipt_exceeds_pending", "La cantidad recibida de "+line.ItemName+" supera lo pendiente.")
			return
		}
		if line.PresentationType != "unit" && math.Abs(line.ReceiveQuantity-math.Round(line.ReceiveQuantity)) > 0.000001 {
			fail(w, 400, "invalid_purchase_receipt_quantity", "Paquetes y cajas deben recibirse en cantidades enteras.")
			return
		}
		lines = append(lines, line)
	}

	for _, line := range lines {
		stockQuantity := math.Round(line.ReceiveQuantity*line.UnitsPerPresentation*1000) / 1000
		if _, err = tx.Exec(r.Context(), `
			INSERT INTO purchase_receipt_items(
				organization_id,purchase_receipt_id,purchase_order_item_id,inventory_item_id,
				presentation_id,quantity,presentation_type,units_per_presentation,unit_cost
			)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			s.OrganizationID, receiptID, line.ID, line.InventoryItemID,
			line.PresentationID, line.ReceiveQuantity, line.PresentationType, line.UnitsPerPresentation,line.UnitCost,
		); err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos guardar el detalle de la recepción.")
			return
		}
		if _, err = tx.Exec(r.Context(), `
			INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
			VALUES($1,$2,$3,0,0)
			ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
			s.OrganizationID, s.LocationID, line.InventoryItemID); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos preparar el saldo de inventario.")
			return
		}
		var current,currentAverage float64
		if err = tx.QueryRow(r.Context(), `
			SELECT quantity::float8,average_unit_cost::float8
			FROM stock_balances
			WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
			FOR UPDATE`, s.OrganizationID, s.LocationID, line.InventoryItemID).Scan(&current,&currentAverage); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos bloquear el saldo de inventario.")
			return
		}
		balanceAfter := math.Round((current+stockQuantity)*1000) / 1000
		baseUnitCost:=0.0
		if line.UnitsPerPresentation>0{baseUnitCost=line.UnitCost/line.UnitsPerPresentation}
		averageAfter:=currentAverage
		if balanceAfter>0{averageAfter=math.Round(((current*currentAverage)+(stockQuantity*baseUnitCost))/balanceAfter*10000)/10000}
		if _, err = tx.Exec(r.Context(), `
			UPDATE stock_balances
			SET quantity=$4,average_unit_cost=$5,updated_at=now()
			WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
			s.OrganizationID, s.LocationID, line.InventoryItemID, balanceAfter,averageAfter); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos actualizar el saldo de inventario.")
			return
		}
		note := "Recepción " + receiptCode + " de " + number
		if in.Notes != "" {
			note += " · " + in.Notes
		}
		valueDelta:=math.Round(stockQuantity*baseUnitCost*10000)/10000
		balanceValue:=math.Round(balanceAfter*averageAfter*10000)/10000
		if _, err = tx.Exec(r.Context(), `
			INSERT INTO stock_movements(
				organization_id,location_id,product_id,inventory_item_id,
				movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by,
				unit_cost,value_delta,balance_value_after
			)
			VALUES($1,$2,$3,$4,'entry',$5,$6,'purchase_receipt',$7,$8,$9,$10,$11,$12)`,
			s.OrganizationID, s.LocationID, line.ProductID, line.InventoryItemID,
			stockQuantity, balanceAfter, receiptID, note, s.UserID,baseUnitCost,valueDelta,balanceValue); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
			return
		}
		if _, err = tx.Exec(r.Context(), `
			UPDATE purchase_order_items
			SET received_quantity=received_quantity+$4
			WHERE id=$1 AND purchase_order_id=$2 AND organization_id=$3`,
			line.ID, r.PathValue("id"), s.OrganizationID, line.ReceiveQuantity); err != nil {
			fail(w, 503, "purchase_unavailable", "No pudimos actualizar lo recibido de la orden.")
			return
		}
	}

	var pendingCount int
	if err = tx.QueryRow(r.Context(), `
		SELECT count(*)
		FROM purchase_order_items
		WHERE purchase_order_id=$1 AND organization_id=$2
		  AND received_quantity < quantity`,
		r.PathValue("id"), s.OrganizationID).Scan(&pendingCount); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos calcular lo pendiente de la orden.")
		return
	}
	nextStatus := "partially_received"
	var receivedAt *time.Time
	if pendingCount == 0 {
		nextStatus = "received"
		now := time.Now().UTC()
		receivedAt = &now
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE purchase_orders
		SET status=$4,
		    received_at=CASE WHEN $4='received' THEN now() ELSE received_at END,
		    updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		r.PathValue("id"), s.OrganizationID, s.LocationID, nextStatus); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos actualizar la recepción de la orden.")
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos recibir la orden.")
		return
	}
	a.audit(r, "purchase.receipt_created", "purchase_order", r.PathValue("id"))
	writeJSON(w, 201, map[string]any{
		"id": receiptID,
		"code": receiptCode,
		"purchaseOrderId": r.PathValue("id"),
		"number": number,
		"status": nextStatus,
		"receivedAt": receivedAt,
	})
}
