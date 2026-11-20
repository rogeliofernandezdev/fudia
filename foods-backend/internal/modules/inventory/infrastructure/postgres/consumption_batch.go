package postgres

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/jackc/pgx/v5"
)

// Lecturas y escrituras por lote del consumo de un pedido. Corren dentro de la
// transacción del pedido y, con la base remota, cada consulta es una ida y
// vuelta: el ciclo anterior hacía unas cuatro por insumo mientras mantenía los
// saldos bloqueados. Los bloqueos siguen el orden de id (ORDER BY ... FOR
// UPDATE), igual que el ciclo ordenado al que reemplazan.

func (r *Stock) RecipeIngredientsFor(ctx context.Context, s scope, recipeIDs []string) (map[string][]domain.RecipeIngredient, error) {
	out := map[string][]domain.RecipeIngredient{}
	if len(recipeIDs) == 0 {
		return out, nil
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT recipe_id::text,inventory_item_id::text,quantity::float8,waste_percent::float8
		FROM product_recipe_items
		WHERE recipe_id=ANY($1::uuid[]) AND organization_id=$2
		ORDER BY recipe_id,inventory_item_id`, recipeIDs, s.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var recipeID string
		var v domain.RecipeIngredient
		if err = rows.Scan(&recipeID, &v.ItemID, &v.Quantity, &v.WastePercent); err != nil {
			return nil, err
		}
		out[recipeID] = append(out[recipeID], v)
	}
	return out, rows.Err()
}

// LockBalances crea los saldos del local que falten y bloquea artículo y saldo
// de cada artículo activo, en orden de id. Los inactivos o inexistentes no
// aparecen en el resultado.
func (r *Stock) LockBalances(ctx context.Context, s scope, itemIDs []string) (map[string]domain.InventoryBalanceLock, error) {
	out := map[string]domain.InventoryBalanceLock{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO stock_balances(organization_id,location_id,inventory_item_id,quantity,average_unit_cost)
		SELECT $1,$2,ii.id,0,0
		FROM inventory_items ii
		WHERE ii.id=ANY($3::uuid[]) AND ii.organization_id=$1 AND ii.active
		ORDER BY ii.id
		ON CONFLICT(location_id,inventory_item_id) DO NOTHING
	`, s.OrganizationID, s.LocationID, itemIDs); err != nil {
		return nil, stockError(err)
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT ii.id::text,ii.product_id,COALESCE(p.name,ii.name),ii.unit,sb.quantity::float8,sb.average_unit_cost::float8
		FROM inventory_items ii
		LEFT JOIN products p ON p.id=ii.product_id AND p.organization_id=ii.organization_id
		JOIN stock_balances sb ON sb.organization_id=ii.organization_id AND sb.inventory_item_id=ii.id
		  AND sb.location_id=$2
		WHERE ii.id=ANY($3::uuid[]) AND ii.organization_id=$1 AND ii.active
		ORDER BY ii.id
		FOR UPDATE OF sb,ii
	`, s.OrganizationID, s.LocationID, itemIDs)
	if err != nil {
		return nil, stockError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var v domain.InventoryBalanceLock
		if err = rows.Scan(&id, &v.ProductID, &v.Name, &v.Unit, &v.Quantity, &v.AverageUnitCost); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

// LockPortionsFor bloquea la disponibilidad del día de los productos por
// porciones, en orden de id; los que no tienen porciones cargadas no aparecen.
func (r *Stock) LockPortionsFor(ctx context.Context, s scope, productIDs []string, businessDate string) (map[string]domain.PortionLock, error) {
	out := map[string]domain.PortionLock{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT product_id::text,portion_quantity,sold_quantity,manual_status
		FROM product_availability
		WHERE organization_id=$1 AND location_id=$2 AND product_id=ANY($3::uuid[]) AND business_date=$4::date
		ORDER BY product_id
		FOR UPDATE`, s.OrganizationID, s.LocationID, productIDs, businessDate)
	if err != nil {
		return nil, stockError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var v domain.PortionLock
		if err = rows.Scan(&id, &v.PortionQuantity, &v.SoldQuantity, &v.ManualStatus); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

// LockProductInventories bloquea el artículo de inventario activo de cada
// producto, en orden de id de artículo (el mismo de LockBalances), y devuelve
// producto → artículo.
func (r *Stock) LockProductInventories(ctx context.Context, s scope, productIDs []string) (map[string]string, error) {
	out := map[string]string{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := r.db(ctx).Query(ctx, `
		SELECT product_id::text,id::text
		FROM inventory_items
		WHERE organization_id=$1 AND product_id=ANY($2::uuid[]) AND active
		ORDER BY id
		FOR UPDATE`, s.OrganizationID, productIDs)
	if err != nil {
		return nil, stockError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var productID, itemID string
		if err = rows.Scan(&productID, &itemID); err != nil {
			return nil, err
		}
		out[productID] = itemID
	}
	return out, rows.Err()
}

// ApplyStockWrites guarda porciones vendidas, saldos y movimientos de Kárdex en
// una sola ida y vuelta, en el orden recibido.
func (r *Stock) ApplyStockWrites(ctx context.Context, s scope, w application.StockWrites) error {
	batch := &pgx.Batch{}
	for _, p := range w.Portions {
		batch.Queue(`
			UPDATE product_availability
			SET sold_quantity=$5,updated_at=now()
			WHERE organization_id=$1 AND location_id=$2 AND product_id=$3 AND business_date=$4::date`,
			s.OrganizationID, s.LocationID, p.ProductID, p.BusinessDate, p.SoldQuantity)
	}
	for _, b := range w.Balances {
		batch.Queue(`UPDATE stock_balances SET quantity=$4,average_unit_cost=$5,updated_at=now() WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
			s.OrganizationID, s.LocationID, b.ItemID, b.Quantity, b.AverageCost)
	}
	userID := movementUser(s)
	for _, m := range w.Movements {
		batch.Queue(insertMovementSQL, s.OrganizationID, s.LocationID, m.ProductID, m.ItemID, m.Type, m.Delta, m.Balance, m.SourceType, m.SourceID, m.Note, userID, m.UnitCost, m.ValueDelta, m.BalanceValue)
	}
	if batch.Len() == 0 {
		return nil
	}
	results := r.db(ctx).SendBatch(ctx, batch)
	for range batch.Len() {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}
	return results.Close()
}
