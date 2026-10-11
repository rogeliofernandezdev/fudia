package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
)

func (r *Purchasing) LockReceivingOrder(ctx context.Context, s scope, id string) (string, string, error) {
	var number string
	var status string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT number,status
		FROM purchase_orders
		WHERE id=$1 AND organization_id=$2 AND location_id=$3
		FOR UPDATE`, id, s.OrganizationID, s.LocationID).Scan(&number, &status)
	return number, status, purchaseError(err)
}
func (r *Purchasing) ExistingReceipt(ctx context.Context, s scope, id string, in domain.PurchaseReceiptInput) (string, string, string, error) {
	var existingID string
	var existingCode string
	var existingStatus string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT pr.id,pr.code,po.status
			FROM purchase_receipts pr
			JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id AND po.location_id=pr.location_id
			WHERE pr.organization_id=$1 AND pr.location_id=$2 AND pr.purchase_order_id=$3 AND pr.idempotency_key=$4
		`, s.OrganizationID, s.LocationID, id, in.IdempotencyKey).Scan(&existingID, &existingCode, &existingStatus)
	return existingID, existingCode, existingStatus, purchaseError(err)
}
func (r *Purchasing) CreateReceipt(ctx context.Context, s scope, id string, in domain.PurchaseReceiptInput) (string, string, error) {
	var receiptID string
	var receiptCode string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO purchase_receipts(
			organization_id,location_id,purchase_order_id,notes,created_by,idempotency_key
		)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,''))
		RETURNING id,code`,
		s.OrganizationID, s.LocationID, id, in.Notes, s.UserID, in.IdempotencyKey,
	).Scan(&receiptID, &receiptCode)
	return receiptID, receiptCode, purchaseError(err)
}
func (r *Purchasing) LockReceivingLine(ctx context.Context, s scope, id string, requested domain.PurchaseReceiptLineInput) (domain.ReceivingLine, error) {
	var line domain.ReceivingLine
	err := r.db(ctx).QueryRow(ctx, `
			SELECT poi.id,poi.inventory_item_id,ii.product_id,COALESCE(p.name,ii.name),
			       poi.presentation_id,poi.presentation_type,poi.units_per_presentation::float8,
			       poi.quantity::float8,poi.received_quantity::float8,poi.unit_cost::float8
			FROM purchase_order_items poi
			JOIN inventory_items ii ON ii.id=poi.inventory_item_id AND ii.organization_id=poi.organization_id
			LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
			WHERE poi.id=$1 AND poi.purchase_order_id=$2 AND poi.organization_id=$3
			FOR UPDATE OF poi`,
		requested.PurchaseOrderItemID, id, s.OrganizationID,
	).Scan(
		&line.ID, &line.InventoryItemID, &line.ProductID, &line.ItemName,
		&line.PresentationID, &line.PresentationType, &line.UnitsPerPresentation,
		&line.OrderedQuantity, &line.ReceivedQuantity, &line.UnitCost,
	)
	return line, purchaseError(err)
}
func (r *Purchasing) CreateReceiptItem(ctx context.Context, s scope, receiptID string, line domain.ReceivingLine) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO purchase_receipt_items(
				organization_id,purchase_receipt_id,purchase_order_item_id,inventory_item_id,
				presentation_id,quantity,presentation_type,units_per_presentation,unit_cost
			)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		s.OrganizationID, receiptID, line.ID, line.InventoryItemID,
		line.PresentationID, line.ReceiveQuantity, line.PresentationType, line.UnitsPerPresentation, line.UnitCost,
	)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) ReceiveOrderLine(ctx context.Context, s scope, id string, line domain.ReceivingLine) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			UPDATE purchase_order_items
			SET received_quantity=received_quantity+$4
			WHERE id=$1 AND purchase_order_id=$2 AND organization_id=$3`,
		line.ID, id, s.OrganizationID, line.ReceiveQuantity)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) PendingLines(ctx context.Context, s scope, id string) (int, error) {
	var pendingCount int
	err := r.db(ctx).QueryRow(ctx, `
		SELECT count(*)
		FROM purchase_order_items
		WHERE purchase_order_id=$1 AND organization_id=$2
		  AND received_quantity < quantity`,
		id, s.OrganizationID).Scan(&pendingCount)
	return pendingCount, purchaseError(err)
}
func (r *Purchasing) ReceiveOrderStatus(ctx context.Context, s scope, id string, nextStatus string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE purchase_orders
		SET status=$4,
		    received_at=CASE WHEN $4='received' THEN now() ELSE received_at END,
		    updated_at=now()
		WHERE id=$1 AND organization_id=$2 AND location_id=$3`,
		id, s.OrganizationID, s.LocationID, nextStatus)
	return tag.RowsAffected(), purchaseError(err)
}
