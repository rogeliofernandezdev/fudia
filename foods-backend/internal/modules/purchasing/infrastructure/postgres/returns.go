package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/purchasing/domain"
)

func (r *Purchasing) LockReturnReceipt(ctx context.Context, s scope, id string) (string, string, string, string, error) {
	var receiptID string
	var orderID string
	var supplierID string
	var number string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT pr.id,po.id,po.supplier_id,po.number
		FROM purchase_receipts pr
		JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id AND po.location_id=pr.location_id
		WHERE pr.id=$1 AND pr.organization_id=$2 AND pr.location_id=$3
		FOR UPDATE OF pr,po
	`, id, s.OrganizationID, s.LocationID).Scan(&receiptID, &orderID, &supplierID, &number)
	return receiptID, orderID, supplierID, number, purchaseError(err)
}
func (r *Purchasing) ExistingReturn(ctx context.Context, s scope, receiptID string, in domain.PurchaseReturnInput) (string, string, string, error) {
	var existingID string
	var existingCode string
	var existingKind string
	err := r.db(ctx).QueryRow(ctx, `
			SELECT id,code,kind FROM purchase_returns
			WHERE organization_id=$1 AND location_id=$2 AND purchase_receipt_id=$3 AND idempotency_key=$4
		`, s.OrganizationID, s.LocationID, receiptID, in.IdempotencyKey).Scan(&existingID, &existingCode, &existingKind)
	return existingID, existingCode, existingKind, purchaseError(err)
}
func (r *Purchasing) CreateReturn(ctx context.Context, s scope, supplierID string, orderID string, receiptID string, in domain.PurchaseReturnInput) (string, string, error) {
	var returnID string
	var code string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO purchase_returns(organization_id,location_id,supplier_id,purchase_order_id,purchase_receipt_id,kind,reason,notes,idempotency_key,created_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10) RETURNING id,code
	`, s.OrganizationID, s.LocationID, supplierID, orderID, receiptID, in.Kind, in.Reason, in.Notes, in.IdempotencyKey, s.UserID).Scan(&returnID, &code)
	return returnID, code, purchaseError(err)
}
func (r *Purchasing) LockReturnLine(ctx context.Context, s scope, receiptID string, requested domain.ReturnLineInput) (string, string, string, float64, string, float64, float64, float64, error) {
	var receiptItemID string
	var orderItemID string
	var itemID string
	var receivedQty float64
	var presentationType string
	var units float64
	var unitCost float64
	var alreadyReturned float64
	err := r.db(ctx).QueryRow(ctx, `
			SELECT pri.id,pri.purchase_order_item_id,pri.inventory_item_id,pri.quantity::float8,
			  pri.presentation_type,pri.units_per_presentation::float8,pri.unit_cost::float8,
			  COALESCE((SELECT sum(x.quantity) FROM purchase_return_items x WHERE x.purchase_receipt_item_id=pri.id AND x.organization_id=pri.organization_id),0)::float8
			FROM purchase_receipt_items pri
			WHERE pri.id=$1 AND pri.purchase_receipt_id=$2 AND pri.organization_id=$3
			FOR UPDATE
		`, requested.PurchaseReceiptItemID, receiptID, s.OrganizationID).Scan(&receiptItemID, &orderItemID, &itemID, &receivedQty, &presentationType, &units, &unitCost, &alreadyReturned)
	return receiptItemID, orderItemID, itemID, receivedQty, presentationType, units, unitCost, alreadyReturned, purchaseError(err)
}
func (r *Purchasing) CreateReturnItem(ctx context.Context, s scope, returnID string, receiptItemID string, orderItemID string, itemID string, requested domain.ReturnLineInput, presentationType string, units float64, unitCost float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO purchase_return_items(organization_id,purchase_return_id,purchase_receipt_item_id,purchase_order_item_id,inventory_item_id,quantity,presentation_type,units_per_presentation,unit_cost)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`, s.OrganizationID, returnID, receiptItemID, orderItemID, itemID, requested.Quantity, presentationType, units, unitCost)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) CorrectReceivedQuantity(ctx context.Context, s scope, orderItemID string, orderID string, requested domain.ReturnLineInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE purchase_order_items SET received_quantity=GREATEST(received_quantity-$4,0) WHERE id=$1 AND purchase_order_id=$2 AND organization_id=$3`, orderItemID, orderID, s.OrganizationID, requested.Quantity)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) ReturnOrderCounts(ctx context.Context, s scope, orderID string) (int, int, error) {
	var receivedLines int
	var pendingLines int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FILTER(WHERE received_quantity>0),count(*) FILTER(WHERE received_quantity<quantity) FROM purchase_order_items WHERE purchase_order_id=$1 AND organization_id=$2`, orderID, s.OrganizationID).Scan(&receivedLines, &pendingLines)
	return receivedLines, pendingLines, purchaseError(err)
}
func (r *Purchasing) ReturnOrderStatus(ctx context.Context, s scope, orderID string, next string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE purchase_orders SET status=$4,received_at=CASE WHEN $4='received' THEN COALESCE(received_at,now()) ELSE NULL END,updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, orderID, s.OrganizationID, s.LocationID, next)
	return tag.RowsAffected(), purchaseError(err)
}
func (r *Purchasing) LockApprovalOrder(ctx context.Context, s scope, id string) (string, error) {
	var status string
	err := r.db(ctx).QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id=$1 AND organization_id=$2 AND location_id=$3 FOR UPDATE`, id, s.OrganizationID, s.LocationID).Scan(&status)
	return status, purchaseError(err)
}
func (r *Purchasing) ApproveOrder(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE purchase_orders SET status='approved',approved_at=now(),updated_at=now() WHERE id=$1 AND organization_id=$2 AND location_id=$3`, id, s.OrganizationID, s.LocationID)
	return tag.RowsAffected(), purchaseError(err)
}
