package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/application"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type scope = requestscope.Scope
type Purchasing struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Purchasing { return &Purchasing{pool} }
func (r *Purchasing) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}
func purchaseError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.PurchaseRepository = (*Purchasing)(nil)

func (r *Purchasing) UpdateSupplier(ctx context.Context, s scope, id string, in domain.SupplierInput) (domain.SupplierView, error) {
	var item domain.SupplierView
	err := r.db(ctx).QueryRow(ctx, `
			UPDATE suppliers
			SET tax_id=NULLIF($3,''),name=$4,email=NULLIF($5,''),phone=NULLIF($6,'')
			WHERE id=$1 AND organization_id=$2
			RETURNING id,COALESCE(tax_id,''),name,COALESCE(email,''),COALESCE(phone,''),active`,
		id, s.OrganizationID, in.TaxID, in.Name, in.Email, in.Phone,
	).Scan(&item.ID, &item.TaxID, &item.Name, &item.Email, &item.Phone, &item.Active)
	return item, purchaseError(err)
}
func (r *Purchasing) CreateSupplier(ctx context.Context, s scope, in domain.SupplierInput) (domain.SupplierView, error) {
	var item domain.SupplierView
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO suppliers(organization_id,tax_id,name,email,phone)
			VALUES($1,NULLIF($2,''),$3,NULLIF($4,''),NULLIF($5,''))
			RETURNING id,COALESCE(tax_id,''),name,COALESCE(email,''),COALESCE(phone,''),active`,
		s.OrganizationID, in.TaxID, in.Name, in.Email, in.Phone,
	).Scan(&item.ID, &item.TaxID, &item.Name, &item.Email, &item.Phone, &item.Active)
	return item, purchaseError(err)
}
func (r *Purchasing) SupplierHasOpenOrders(ctx context.Context, s scope, id string) (bool, error) {
	var open bool
	err := r.db(ctx).QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM purchase_orders
				WHERE organization_id=$1 AND supplier_id=$2
				  AND status IN ('draft','pending_approval','approved','partially_received')
			)`, s.OrganizationID, id).Scan(&open)
	return open, purchaseError(err)
}
func (r *Purchasing) SetSupplierActive(ctx context.Context, s scope, id string, in struct{ Active bool }) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE suppliers SET active=$3
		WHERE id=$1 AND organization_id=$2`,
		id, s.OrganizationID, in.Active)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) SupplierActive(ctx context.Context, s scope, in domain.PurchaseOrderInput) (bool, error) {
	var supplierActive bool
	err := r.db(ctx).QueryRow(ctx, `
		SELECT active FROM suppliers WHERE id=$1 AND organization_id=$2`,
		in.SupplierID, s.OrganizationID).Scan(&supplierActive)
	return supplierActive, purchaseError(err)
}
func (r *Purchasing) LockOrder(ctx context.Context, s scope, id string) (string, string, error) {
	var number string
	var status string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT number,status
			FROM purchase_orders
			WHERE id=$1 AND organization_id=$2 AND location_id=$3
			FOR UPDATE`, id, s.OrganizationID, s.LocationID).Scan(&number, &status)
	return number, status, purchaseError(err)
}
func (r *Purchasing) UpdateOrder(ctx context.Context, s scope, id string, in domain.PurchaseOrderInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			UPDATE purchase_orders
			SET supplier_id=$4,expected_at=NULLIF($5,'')::date,notes=$6,total=0,updated_at=now()
			WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		id, s.OrganizationID, s.LocationID, in.SupplierID, in.ExpectedAt, in.Notes)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) DeleteOrderItems(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			DELETE FROM purchase_order_items
			WHERE purchase_order_id=$1 AND organization_id=$2`, id, s.OrganizationID)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) CreateOrder(ctx context.Context, s scope, in domain.PurchaseOrderInput) (string, string, error) {
	var id string
	var number string
	err := r.db(ctx).QueryRow(ctx, `
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
	return id, number, purchaseError(err)
}
func (r *Purchasing) OrderPresentation(ctx context.Context, s scope, line domain.PurchaseOrderItemInput) (string, float64, error) {
	var presentationType string
	var unitsPerPresentation float64
	err := r.db(ctx).QueryRow(ctx, `
			SELECT ip.presentation_type,ip.units_per_presentation::float8
			FROM inventory_presentations ip
			JOIN inventory_items ii
			  ON ii.id=ip.inventory_item_id AND ii.organization_id=ip.organization_id
			WHERE ip.id=$1 AND ip.inventory_item_id=$2 AND ip.organization_id=$3
			  AND ip.active AND ii.active`,
		line.PresentationID, line.InventoryItemID, s.OrganizationID,
	).Scan(&presentationType, &unitsPerPresentation)
	return presentationType, unitsPerPresentation, purchaseError(err)
}
func (r *Purchasing) CreateOrderItem(ctx context.Context, s scope, id string, line domain.PurchaseOrderItemInput, presentationType string, unitsPerPresentation float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO purchase_order_items(
				organization_id,purchase_order_id,inventory_item_id,presentation_id,
				quantity,presentation_type,units_per_presentation,unit_cost
			)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8::numeric)`,
		s.OrganizationID, id, line.InventoryItemID, line.PresentationID,
		line.Quantity, presentationType, unitsPerPresentation, line.UnitCost,
	)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) RecalculateOrder(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE purchase_orders
		SET total=COALESCE((
			SELECT sum(line_total)
			FROM purchase_order_items
			WHERE purchase_order_id=$1 AND organization_id=$2
		),0),updated_at=now()
		WHERE id=$1 AND organization_id=$2`, id, s.OrganizationID)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) LockOrderStatus(ctx context.Context, s scope, id string) (string, error) {
	var current string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT status FROM purchase_orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE`, id, s.OrganizationID, s.LocationID).Scan(&current)
	return current, purchaseError(err)
}
func (r *Purchasing) SetOrderStatus(ctx context.Context, s scope, id string, in application.OrderStatusInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE purchase_orders
		SET status=$4,updated_at=now(),
		    approved_at=CASE WHEN $4='approved' THEN now() WHEN $4='draft' THEN NULL ELSE approved_at END,
		    cancelled_at=CASE WHEN $4='cancelled' THEN now() ELSE cancelled_at END
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		id, s.OrganizationID, s.LocationID, in.Status)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) Suppliers(ctx context.Context, s scope, f application.Filter) (application.SuppliersPage, error) {
	out := application.SuppliersPage{Page: f.Page, PageSize: f.Size}
	q, page, size := strings.TrimSpace(f.Query), f.Page, f.Size
	status := strings.TrimSpace(f.Status)
	where := `organization_id=$1
		AND ($2='' OR name ILIKE '%'||$2||'%' OR COALESCE(tax_id,'') ILIKE '%'||$2||'%')
		AND ($3='' OR ($3='active' AND active) OR ($3='inactive' AND NOT active))`
	var total int
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM suppliers WHERE `+where, s.OrganizationID, q, status).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT id,COALESCE(tax_id,''),name,COALESCE(email,''),COALESCE(phone,''),active
		FROM suppliers
		WHERE `+where+`
		ORDER BY active DESC,name
		LIMIT $4 OFFSET $5`, s.OrganizationID, q, status, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.SupplierView{}
	for rows.Next() {
		var item domain.SupplierView
		if err := rows.Scan(&item.ID, &item.TaxID, &item.Name, &item.Email, &item.Phone, &item.Active); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Purchasing) Orders(ctx context.Context, s scope, f application.Filter) (application.OrdersPage, error) {
	out := application.OrdersPage{Page: f.Page, PageSize: f.Size}
	q, page, size := strings.TrimSpace(f.Query), f.Page, f.Size
	status := strings.TrimSpace(f.Status)
	where := `po.organization_id=$1 AND po.location_id=$2
		AND ($3='' OR po.number ILIKE '%'||$3||'%' OR sp.name ILIKE '%'||$3||'%')
		AND (
		  $4=''
		  OR ($4='receivable' AND po.status IN ('approved','partially_received'))
		  OR po.status=$4
		)`
	var total int
	if err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*)
		FROM purchase_orders po
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE `+where, s.OrganizationID, s.LocationID, q, status).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
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
		return out, err
	}
	defer rows.Close()
	items := []domain.PurchaseOrderSummary{}
	for rows.Next() {
		var item domain.PurchaseOrderSummary
		if err := rows.Scan(
			&item.ID, &item.Number, &item.SupplierID, &item.SupplierName, &item.Status,
			&item.Total, &item.Notes, &item.ExpectedAt, &item.ItemCount, &item.CreatedAt,
			&item.UpdatedAt, &item.ApprovedAt, &item.ReceivedAt, &item.CancelledAt,
		); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Purchasing) Receipts(ctx context.Context, s scope, f application.Filter) (application.ReceiptsPage, error) {
	out := application.ReceiptsPage{Page: f.Page, PageSize: f.Size}
	q, page, size := strings.TrimSpace(f.Query), f.Page, f.Size
	from, to := strings.TrimSpace(f.From), strings.TrimSpace(f.To)
	where := `pr.organization_id=$1 AND pr.location_id=$2
	  AND ($3='' OR pr.code ILIKE '%'||$3||'%' OR po.number ILIKE '%'||$3||'%' OR sp.name ILIKE '%'||$3||'%')
	  AND ($4='' OR pr.created_at >= ($4::date::timestamp AT TIME ZONE (SELECT timezone FROM locations WHERE id=$2 AND organization_id=$1)))
	  AND ($5='' OR pr.created_at < (($5::date + interval '1 day')::timestamp AT TIME ZONE (SELECT timezone FROM locations WHERE id=$2 AND organization_id=$1)))`
	var total int
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE `+where, s.OrganizationID, s.LocationID, q, from, to).Scan(&total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT pr.id,pr.code,po.id,po.number,sp.name,pr.notes,COALESCE(u.full_name,'Usuario no disponible'),pr.created_at,
		  (SELECT count(*) FROM purchase_receipt_items pri WHERE pri.purchase_receipt_id=pr.id AND pri.organization_id=pr.organization_id)
		FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		LEFT JOIN users u ON u.id=pr.created_by
		WHERE `+where+` ORDER BY pr.created_at DESC LIMIT $6 OFFSET $7
	`, s.OrganizationID, s.LocationID, q, from, to, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.PurchaseReceiptSummary{}
	for rows.Next() {
		var x domain.PurchaseReceiptSummary
		if err := rows.Scan(&x.ID, &x.Code, &x.PurchaseOrderID, &x.Number, &x.SupplierName, &x.Notes, &x.CreatedByName, &x.CreatedAt, &x.ItemCount); err != nil {
			return out, err
		}
		items = append(items, x)
	}
	out.Items = items
	out.Total = total
	return out, rows.Err()
}
func (r *Purchasing) Order(ctx context.Context, s scope, id string) (domain.PurchaseOrderDetail, error) {
	var item domain.PurchaseOrderDetail
	err := r.db(ctx).QueryRow(ctx, `
		SELECT po.id,po.number,po.supplier_id,sp.name,po.status,po.total::text,po.notes,
		       CASE WHEN po.expected_at IS NULL THEN NULL ELSE po.expected_at::text END,
		       (SELECT count(*) FROM purchase_order_items poi WHERE poi.organization_id=po.organization_id AND poi.purchase_order_id=po.id),
		       po.created_at,po.updated_at,po.approved_at,po.received_at,po.cancelled_at
		FROM purchase_orders po
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		WHERE po.id=$1 AND po.organization_id=$2 AND po.location_id=$3`,
		id, s.OrganizationID, s.LocationID,
	).Scan(
		&item.ID, &item.Number, &item.SupplierID, &item.SupplierName, &item.Status,
		&item.Total, &item.Notes, &item.ExpectedAt, &item.ItemCount, &item.CreatedAt,
		&item.UpdatedAt, &item.ApprovedAt, &item.ReceivedAt, &item.CancelledAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, purchaseError(err)
	}
	if err != nil {
		return item, purchaseError(err)
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT poi.id,poi.inventory_item_id,COALESCE(p.name,ii.name),COALESCE(p.sku,ii.sku),ii.unit,
		       poi.presentation_id,poi.presentation_type,poi.units_per_presentation::text,
		       poi.quantity::text,poi.received_quantity::text,(poi.quantity-poi.received_quantity)::text,
		       poi.stock_quantity::text,poi.unit_cost::text,poi.line_total::text,pt.name
		FROM purchase_order_items poi
		JOIN inventory_presentation_types pt ON pt.organization_id=poi.organization_id AND pt.code=poi.presentation_type
		JOIN inventory_items ii ON ii.id=poi.inventory_item_id AND ii.organization_id=poi.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE poi.purchase_order_id=$1 AND poi.organization_id=$2
		ORDER BY poi.created_at,COALESCE(p.name,ii.name)`,
		item.ID, s.OrganizationID)
	if err != nil {
		return item, purchaseError(err)
	}
	defer rows.Close()
	item.Items = []domain.PurchaseOrderItemView{}
	for rows.Next() {
		var line domain.PurchaseOrderItemView
		if err := rows.Scan(
			&line.ID, &line.InventoryItemID, &line.ItemName, &line.SKU, &line.Unit,
			&line.PresentationID, &line.PresentationType, &line.UnitsPerPresentation,
			&line.Quantity, &line.ReceivedQuantity, &line.PendingQuantity,
			&line.StockQuantity, &line.UnitCost, &line.LineTotal, &line.PresentationName,
		); err != nil {
			return item, purchaseError(err)
		}
		item.Items = append(item.Items, line)
	}
	if err := rows.Err(); err != nil {
		return item, purchaseError(err)
	}
	item.History, err = r.orderHistory(ctx, s, item.ID)
	return item, err
}

// orderHistory lists the receipts of an order and the corrections/returns made
// against them, in the order they happened.
func (r *Purchasing) orderHistory(ctx context.Context, s scope, orderID string) ([]domain.PurchaseOrderEvent, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT id,kind,code,receipt_id,reason,notes,created_by_name,created_at,item_count FROM (
		  SELECT pr.id,'receipt' AS kind,pr.code,pr.id AS receipt_id,'' AS reason,pr.notes,
		         COALESCE(u.full_name,'Usuario no disponible') AS created_by_name,pr.created_at,
		         (SELECT count(*) FROM purchase_receipt_items pri WHERE pri.purchase_receipt_id=pr.id AND pri.organization_id=pr.organization_id)::int AS item_count
		  FROM purchase_receipts pr
		  LEFT JOIN users u ON u.id=pr.created_by
		  WHERE pr.purchase_order_id=$1 AND pr.organization_id=$2 AND pr.location_id=$3
		  UNION ALL
		  SELECT rt.id,rt.kind,rt.code,rt.purchase_receipt_id,rt.reason,rt.notes,
		         COALESCE(u.full_name,'Usuario no disponible'),rt.created_at,
		         (SELECT count(*) FROM purchase_return_items ri WHERE ri.purchase_return_id=rt.id AND ri.organization_id=rt.organization_id)::int
		  FROM purchase_returns rt
		  LEFT JOIN users u ON u.id=rt.created_by
		  WHERE rt.purchase_order_id=$1 AND rt.organization_id=$2 AND rt.location_id=$3
		) history ORDER BY created_at,kind<>'receipt'`, orderID, s.OrganizationID, s.LocationID)
	if err != nil {
		return nil, purchaseError(err)
	}
	defer rows.Close()
	events := []domain.PurchaseOrderEvent{}
	for rows.Next() {
		var e domain.PurchaseOrderEvent
		if err := rows.Scan(&e.ID, &e.Kind, &e.Code, &e.ReceiptID, &e.Reason, &e.Notes, &e.CreatedByName, &e.CreatedAt, &e.ItemCount); err != nil {
			return nil, purchaseError(err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
func (r *Purchasing) Receipt(ctx context.Context, s scope, id string) (domain.PurchaseReceiptDetail, error) {
	var out domain.PurchaseReceiptDetail
	err := r.db(ctx).QueryRow(ctx, `
		SELECT pr.id,pr.code,po.id,po.number,sp.name,pr.notes,COALESCE(u.full_name,'Usuario no disponible'),pr.created_at,
		  (SELECT count(*) FROM purchase_receipt_items pri WHERE pri.purchase_receipt_id=pr.id AND pri.organization_id=pr.organization_id)
		FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
		JOIN suppliers sp ON sp.id=po.supplier_id AND sp.organization_id=po.organization_id
		LEFT JOIN users u ON u.id=pr.created_by
		WHERE pr.id=$1 AND pr.organization_id=$2 AND pr.location_id=$3
	`, id, s.OrganizationID, s.LocationID).Scan(&out.ID, &out.Code, &out.PurchaseOrderID, &out.Number, &out.SupplierName, &out.Notes, &out.CreatedByName, &out.CreatedAt, &out.ItemCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, purchaseError(err)
	}
	if err != nil {
		return out, purchaseError(err)
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT pri.id,pri.purchase_order_item_id,pri.inventory_item_id,COALESCE(p.name,ii.name),
		  pri.quantity::text,pri.presentation_type,pri.units_per_presentation::text,pri.stock_quantity::text,pri.unit_cost::text,pt.name,
		  COALESCE((SELECT sum(x.quantity) FROM purchase_return_items x WHERE x.purchase_receipt_item_id=pri.id AND x.organization_id=pri.organization_id),0)::text,
		  GREATEST(pri.quantity-COALESCE((SELECT sum(x.quantity) FROM purchase_return_items x WHERE x.purchase_receipt_item_id=pri.id AND x.organization_id=pri.organization_id),0),0)::text
		FROM purchase_receipt_items pri
		JOIN inventory_presentation_types pt ON pt.organization_id=pri.organization_id AND pt.code=pri.presentation_type
		JOIN inventory_items ii ON ii.id=pri.inventory_item_id AND ii.organization_id=pri.organization_id
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		WHERE pri.purchase_receipt_id=$1 AND pri.organization_id=$2 ORDER BY pri.created_at,pri.id
	`, out.ID, s.OrganizationID)
	if err != nil {
		return out, purchaseError(err)
	}
	defer rows.Close()
	out.Items = []domain.PurchaseReceiptItemView{}
	for rows.Next() {
		var x domain.PurchaseReceiptItemView
		if err := rows.Scan(&x.ID, &x.PurchaseOrderItemID, &x.InventoryItemID, &x.ItemName, &x.Quantity, &x.PresentationType, &x.UnitsPerPresentation, &x.StockQuantity, &x.UnitCost, &x.PresentationName, &x.ReturnedQuantity, &x.ReturnableQuantity); err != nil {
			return out, purchaseError(err)
		}
		out.Items = append(out.Items, x)
	}
	return out, rows.Err()
}
