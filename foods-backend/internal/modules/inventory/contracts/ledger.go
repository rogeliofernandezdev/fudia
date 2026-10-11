// Package contracts exposes Inventory's integration payloads without leaking its application layer.
package contracts

import "github.com/foods-platform/foods-backend/internal/modules/inventory/domain"

type Balance = domain.InventoryBalanceLock

type Movement struct {
	ProductID                                          *string
	ItemID, Type, SourceType, SourceID, Note           string
	Delta, Balance, UnitCost, ValueDelta, BalanceValue float64
}
