package postgres

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

func (r *Stock) TransferDestination(ctx context.Context, s scope, in domain.TransferInput) (string, string, error) {
	var id, destination string
	err := r.db(ctx).QueryRow(ctx, `SELECT id::text,name FROM locations WHERE id=$1 AND organization_id=$2 AND active FOR SHARE`, in.ToLocationID, s.OrganizationID).Scan(&id, &destination)
	return id, destination, stockError(err)
}
func (r *Stock) ExistingTransfer(ctx context.Context, s scope, in domain.TransferInput) (string, string, string, error) {
	var existingID string
	var existingCode string
	var existingTo string
	err := r.db(ctx).QueryRow(ctx, `SELECT id,code,to_location_id::text FROM inventory_transfers WHERE organization_id=$1 AND from_location_id=$2 AND idempotency_key=$3`, s.OrganizationID, s.LocationID, in.IdempotencyKey).Scan(&existingID, &existingCode, &existingTo)
	return existingID, existingCode, existingTo, stockError(err)
}
func (r *Stock) CreateTransfer(ctx context.Context, s scope, in domain.TransferInput) (string, string, error) {
	var transferID string
	var code string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO inventory_transfers(organization_id,from_location_id,to_location_id,notes,idempotency_key,created_by) VALUES($1,$2,$3,$4,NULLIF($5,''),$6) RETURNING id,code`, s.OrganizationID, s.LocationID, in.ToLocationID, in.Notes, in.IdempotencyKey, s.UserID).Scan(&transferID, &code)
	return transferID, code, stockError(err)
}
func (r *Stock) TransferItem(ctx context.Context, s scope, line domain.TransferLine) (*string, string, error) {
	var productID *string
	var name string
	err := r.db(ctx).QueryRow(ctx, `SELECT product_id,COALESCE((SELECT name FROM products p WHERE p.id=inventory_items.product_id),name) FROM inventory_items WHERE id=$1 AND organization_id=$2 AND active FOR SHARE`, line.InventoryItemID, s.OrganizationID).Scan(&productID, &name)
	return productID, name, stockError(err)
}
func (r *Stock) EnsureTransferBalances(ctx context.Context, s scope, line domain.TransferLine, in domain.TransferInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
			VALUES($1,$2,$3,0,0),($1,$4,$3,0,0)
			ON CONFLICT(location_id,inventory_item_id) DO NOTHING
		`, s.OrganizationID, s.LocationID, line.InventoryItemID, in.ToLocationID)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) TransferSourceBalance(ctx context.Context, s scope, line domain.TransferLine, srcAfter float64, srcAvg float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, s.LocationID, line.InventoryItemID, srcAfter, srcAvg)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) TransferDestinationBalance(ctx context.Context, s scope, line domain.TransferLine, in domain.TransferInput, dstAfter float64, dstAvg float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`, s.OrganizationID, in.ToLocationID, line.InventoryItemID, dstAfter, dstAvg)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) CreateTransferItem(ctx context.Context, s scope, transferID string, line domain.TransferLine, src domain.LocationBalance) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO inventory_transfer_items(organization_id,transfer_id,inventory_item_id,quantity,unit_cost) VALUES($1,$2,$3,$4,$5)`, s.OrganizationID, transferID, line.InventoryItemID, line.Quantity, src.Cost)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) TransferIncomingMovement(ctx context.Context, s scope, in domain.TransferInput, productID *string, line domain.TransferLine, dstAfter float64, transferID string, code string, src domain.LocationBalance, valueDelta float64, balanceValue float64) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
			INSERT INTO stock_movements(organization_id,location_id,product_id,inventory_item_id,movement_type,quantity_delta,balance_after,source_type,source_id,note,created_by,unit_cost,value_delta,balance_value_after)
			VALUES($1,$2,$3,$4,'transfer_in',$5,$6,'inventory_transfer',$7,$8,$9,$10,$11,$12)
		`, s.OrganizationID, in.ToLocationID, productID, line.InventoryItemID, line.Quantity, dstAfter, transferID, code+" desde transferencia", s.UserID, src.Cost, valueDelta, balanceValue)
	return tag.RowsAffected(), stockError(err)
}
func (r *Stock) LockTransferBalances(ctx context.Context, s scope, itemID, destination string) (map[string]domain.LocationBalance, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT location_id,quantity::float8,average_unit_cost::float8 FROM stock_balances WHERE organization_id=$1 AND inventory_item_id=$2 AND location_id IN ($3,$4) ORDER BY location_id FOR UPDATE`, s.OrganizationID, itemID, s.LocationID, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]domain.LocationBalance{}
	for rows.Next() {
		var id string
		var v domain.LocationBalance
		if err = rows.Scan(&id, &v.Quantity, &v.Cost); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

func (r *Stock) Transfers(ctx context.Context, s scope, f application.StockFilter) (application.TransfersPage, error) {
	out := application.TransfersPage{Items: []domain.TransferSummary{}, Page: f.Page, PageSize: f.Size}
	if err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM inventory_transfers WHERE organization_id=$1 AND (from_location_id=$2 OR to_location_id=$2)`, s.OrganizationID, s.LocationID).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT t.id,t.code,fl.name,tl.name,t.notes,COALESCE(u.full_name,'Usuario no disponible'),t.created_at,
		  (SELECT count(*) FROM inventory_transfer_items x WHERE x.transfer_id=t.id AND x.organization_id=t.organization_id)
		FROM inventory_transfers t JOIN locations fl ON fl.id=t.from_location_id JOIN locations tl ON tl.id=t.to_location_id
		LEFT JOIN users u ON u.id=t.created_by
		WHERE t.organization_id=$1 AND (t.from_location_id=$2 OR t.to_location_id=$2)
		ORDER BY t.created_at DESC LIMIT $3 OFFSET $4
	`, s.OrganizationID, s.LocationID, f.Size, (f.Page-1)*f.Size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.TransferSummary
		if err = rows.Scan(&v.ID, &v.Code, &v.FromLocationName, &v.ToLocationName, &v.Notes, &v.CreatedByName, &v.CreatedAt, &v.ItemCount); err != nil {
			return out, err
		}
		out.Items = append(out.Items, v)
	}
	return out, rows.Err()
}
