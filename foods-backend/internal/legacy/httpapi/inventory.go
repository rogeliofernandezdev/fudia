package httpapi

import (
	"context"
	"errors"
	inventorydomain "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type inventoryItemView = inventorydomain.InventoryItemView

type inventoryPresentationOption = inventorydomain.InventoryPresentationOption

type inventoryProductOption = inventorydomain.InventoryProductOption

type inventoryNewProductInput = inventorydomain.InventoryNewProductInput

type inventoryNewIngredientInput = inventorydomain.InventoryNewIngredientInput

type inventoryEntryInput = inventorydomain.InventoryEntryInput

type inventoryCatalogCreateResult = inventorydomain.InventoryCatalogCreateResult

type inventoryMovementView = inventorydomain.InventoryMovementView

type inventoryAdjustmentInput = inventorydomain.InventoryAdjustmentInput

var normalizeInventoryEntry = inventorydomain.NormalizeInventoryEntry

func (a *API) listInventory(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventory(w, r)
}

func (a *API) listInventoryProducts(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryProducts(w, r)
}

func (a *API) createInventoryEntry(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryEntry(w, r)
}

var validInventoryAdjustmentReason = inventorydomain.ValidInventoryAdjustmentReason

var normalizeInventoryAdjustment = inventorydomain.NormalizeInventoryAdjustment

func (a *API) createInventoryAdjustment(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.CreateInventoryAdjustment(w, r)
}

func (a *API) listInventoryMovements(w http.ResponseWriter, r *http.Request) {
	a.handlers.InventoryStock.ListInventoryMovements(w, r)
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
	for id := range previous {
		ids[id] = true
	}
	for id := range next {
		ids[id] = true
	}
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
	for id := range delta {
		productIDs = append(productIDs, id)
	}
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
			if step > 0 && manualStatus == "sold_out" {
				return &orderPreparationError{Status: 409, Code: "insufficient_portions", Message: name + " está agotado por hoy."}
			}
			if step > 0 && (portionQuantity == nil || nextSold > *portionQuantity) {
				left := 0
				if portionQuantity != nil && *portionQuantity > soldQuantity {
					left = *portionQuantity - soldQuantity
				}
				return &orderPreparationError{Status: 409, Code: "insufficient_portions", Message: "La cantidad excede las existencias: solo quedan " + strconv.Itoa(left) + " porciones de " + name + "."}
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
			var balance, averageCost float64
			if err := tx.QueryRow(ctx, `
				SELECT quantity::float8,average_unit_cost::float8
				FROM stock_balances
				WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3
				FOR UPDATE`, s.OrganizationID, s.LocationID, inventoryItemID).Scan(&balance, &averageCost); err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos bloquear el saldo del producto."}
			}
			balanceDelta := -change
			nextBalance := balance + balanceDelta
			if nextBalance < -0.000001 {
				left := strconv.FormatFloat(math.Max(0, balance), 'f', -1, 64)
				return &orderPreparationError{Status: 409, Code: "insufficient_stock", Message: "La cantidad excede las existencias: solo quedan " + left + " unidades de " + name + "."}
			}
			if nextBalance < 0 {
				nextBalance = 0
			}
			nextAverage := averageCost
			if _, err := tx.Exec(ctx, `
				UPDATE stock_balances
				SET quantity=$4,average_unit_cost=$5,updated_at=now()
				WHERE organization_id=$1 AND location_id=$2 AND inventory_item_id=$3`,
				s.OrganizationID, s.LocationID, inventoryItemID, nextBalance, nextAverage); err != nil {
				return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos actualizar el stock del producto."}
			}
			if math.Abs(balanceDelta) > 0.000001 {
				if err := insertValuedMovement(ctx, tx, s, &productID, inventoryItemID, movementType, balanceDelta, nextBalance, averageCost, "order", orderID, "Pedido "+orderID); err != nil {
					return &orderPreparationError{Status: 503, Code: "quantity_unavailable", Message: "No pudimos registrar el movimiento de Kárdex."}
				}
			}
		default:
			return &orderPreparationError{Status: 503, Code: "quantity_control_invalid", Message: "El producto tiene un control de cantidad inválido."}
		}
	}
	return nil
}

// validDateFilters accepts empty values or ISO dates (YYYY-MM-DD) for range filters.
var validDateFilters = inventorydomain.ValidDateFilters
