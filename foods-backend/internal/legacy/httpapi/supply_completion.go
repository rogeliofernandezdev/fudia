package httpapi

import (
	"context"
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"math"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

type inventoryBalanceLock = inventorydomain.InventoryBalanceLock

func lockInventoryBalance(ctx context.Context, tx pgx.Tx, s scope, inventoryItemID string) (inventoryBalanceLock, error) {
	var out inventoryBalanceLock
	if _, err := tx.Exec(ctx, `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
		SELECT $1,$2,ii.id,0,0
		FROM inventory_items ii
		WHERE ii.id=$3 AND ii.organization_id=$1 AND ii.active
		ON CONFLICT(location_id,inventory_item_id) DO NOTHING
	`, s.OrganizationID, s.LocationID, inventoryItemID); err != nil {
		return out, err
	}
	err := tx.QueryRow(ctx, `
		SELECT ii.product_id,COALESCE(p.name,ii.name),ii.unit,sb.quantity::float8,sb.average_unit_cost::float8
		FROM inventory_items ii
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		JOIN stock_balances sb ON sb.organization_id=ii.organization_id AND sb.inventory_item_id=ii.id
		  AND sb.location_id=$2
		WHERE ii.id=$3 AND ii.organization_id=$1 AND ii.active
		FOR UPDATE OF sb,ii
	`, s.OrganizationID, s.LocationID, inventoryItemID).Scan(&out.ProductID, &out.Name, &out.Unit, &out.Quantity, &out.AverageUnitCost)
	return out, err
}

func nullableUserID(id string) any {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	return id
}

func insertValuedMovement(ctx context.Context, tx pgx.Tx, s scope, productID *string, inventoryItemID, movementType string, delta, balanceAfter, unitCost float64, sourceType, sourceID, note string) error {
	valueDelta := math.Round(delta*unitCost*10000) / 10000
	balanceValue := math.Round(balanceAfter*unitCost*10000) / 10000
	_, err := tx.Exec(ctx, `
		INSERT INTO stock_movements(
		  organization_id,location_id,product_id,inventory_item_id,movement_type,
		  quantity_delta,balance_after,source_type,source_id,note,created_by,
		  unit_cost,value_delta,balance_value_after
		)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, s.OrganizationID, s.LocationID, productID, inventoryItemID, movementType, delta, balanceAfter, sourceType, sourceID, note, nullableUserID(s.UserID), unitCost, valueDelta, balanceValue)
	return err
}

func (a *API) updateInventoryLocationSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.UpdateInventoryLocationSettings(w, r)
}

type transferInput = inventorydomain.TransferInput

func (a *API) createInventoryTransfer(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryTransfer(w, r)
}

func (a *API) listInventoryTransfers(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryTransfers(w, r)
}
