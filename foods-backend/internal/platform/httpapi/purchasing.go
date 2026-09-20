package httpapi

import (
	"encoding/json"
	"errors"
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
				  AND status IN ('draft','pending_approval','approved')
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

func (a *API) listPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	s := r.Context().Value(scopeKey{}).(scope)
	page, size := pageParams(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	where := `po.organization_id=$1 AND po.location_id=$2
		AND ($3='' OR po.number ILIKE '%'||$3||'%' OR sp.name ILIKE '%'||$3||'%')
		AND ($4='' OR po.status=$4)`
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
		       poi.quantity::text,poi.stock_quantity::text,poi.unit_cost::text,poi.line_total::text
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
			&line.Quantity, &line.StockQuantity, &line.UnitCost, &line.LineTotal,
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
		"pending_approval": {"draft": true, "approved": true, "cancelled": true},
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
	if status != "approved" {
		fail(w, 409, "purchase_not_approved", "Solo una orden aprobada puede recibirse.")
		return
	}

	type receivingLine struct {
		InventoryItemID      string
		ProductID            *string
		ItemName             string
		Unit                 string
		PresentationID       string
		PresentationType     string
		UnitsPerPresentation float64
		Quantity             float64
		StockQuantity        float64
	}
	rows, err := tx.Query(r.Context(), `
		SELECT poi.inventory_item_id,ii.product_id,COALESCE(p.name,ii.name),ii.unit,
		       poi.presentation_id,poi.presentation_type,poi.units_per_presentation::float8,
		       poi.quantity::float8,poi.stock_quantity::float8
		FROM purchase_order_items poi
		JOIN inventory_items ii ON ii.id=poi.inventory_item_id AND ii.organization_id=poi.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE poi.purchase_order_id=$1 AND poi.organization_id=$2
		ORDER BY poi.created_at,poi.id`, r.PathValue("id"), s.OrganizationID)
	if err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos cargar los artículos a recibir.")
		return
	}
	lines := []receivingLine{}
	for rows.Next() {
		var line receivingLine
		if err := rows.Scan(
			&line.InventoryItemID, &line.ProductID, &line.ItemName, &line.Unit,
			&line.PresentationID, &line.PresentationType, &line.UnitsPerPresentation,
			&line.Quantity, &line.StockQuantity,
		); err != nil {
			rows.Close()
			fail(w, 503, "purchase_unavailable", "No pudimos preparar la recepción.")
			return
		}
		lines = append(lines, line)
	}
	rows.Close()
	if len(lines) == 0 {
		fail(w, 409, "purchase_empty", "La orden no tiene artículos para recibir.")
		return
	}

	for _, line := range lines {
		if _, err = tx.Exec(r.Context(), `
			INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity)
			VALUES($1,$2,$3,0)
			ON CONFLICT (location_id,inventory_item_id) DO NOTHING`,
			s.OrganizationID, s.LocationID, line.InventoryItemID); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos preparar el saldo de inventario.")
			return
		}
		var current float64
		if err = tx.QueryRow(r.Context(), `
			SELECT quantity::float8
			FROM stock_balances
			WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
			FOR UPDATE`, s.OrganizationID, s.LocationID, line.InventoryItemID).Scan(&current); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos bloquear el saldo de inventario.")
			return
		}
		balanceAfter := current + line.StockQuantity
		if _, err = tx.Exec(r.Context(), `
			UPDATE stock_balances
			SET quantity=$4,updated_at=now()
			WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
			s.OrganizationID, s.LocationID, line.InventoryItemID, balanceAfter); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos actualizar el saldo de inventario.")
			return
		}
		var entryID string
		note := "Recepción de " + number
		if err = tx.QueryRow(r.Context(), `
			INSERT INTO inventory_entries(
				organization_id,location_id,product_id,inventory_item_id,
				quantity,unit,presentation_id,presentation_type,units_per_presentation,note,created_by
			)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
			RETURNING id`,
			s.OrganizationID, s.LocationID, line.ProductID, line.InventoryItemID,
			line.Quantity, line.Unit, line.PresentationID, line.PresentationType,
			line.UnitsPerPresentation, note, s.UserID,
		).Scan(&entryID); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos registrar la entrada de inventario.")
			return
		}
		if _, err = tx.Exec(r.Context(), `
			INSERT INTO stock_movements(
				organization_id,location_id,product_id,inventory_item_id,
				movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by
			)
			VALUES($1,$2,$3,$4,'entry',$5,$6,'inventory_entry',$7,$8,$9)`,
			s.OrganizationID, s.LocationID, line.ProductID, line.InventoryItemID,
			line.StockQuantity, balanceAfter, entryID, note, s.UserID); err != nil {
			fail(w, 503, "inventory_unavailable", "No pudimos registrar el movimiento de Kárdex.")
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `
		UPDATE purchase_orders
		SET status='received',received_at=now(),updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		r.PathValue("id"), s.OrganizationID, s.LocationID); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos cerrar la recepción.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, 503, "purchase_unavailable", "No pudimos recibir la orden.")
		return
	}
	a.audit(r, "purchase.received", "purchase_order", r.PathValue("id"))
	writeJSON(w, 200, map[string]any{"id": r.PathValue("id"), "number": number, "status": "received", "receivedAt": time.Now().UTC()})
}
