// Package contracts exposes Inventory's integration payloads without leaking its application layer.
package contracts

import (
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

type Balance = domain.InventoryBalanceLock

type Movement struct {
	ProductID                                          *string
	ItemID, Type, SourceType, SourceID, Note           string
	Delta, Balance, UnitCost, ValueDelta, BalanceValue float64
}

type PurchaseReceipt struct {
	ItemID string
	ProductID *string
	Quantity, UnitsPerPresentation, UnitCost float64
	ReceiptID, Note string
}

type PurchaseReturn struct {
	ItemID, Kind, ReturnID, Note string
	Quantity, UnitsPerPresentation float64
}

// ErrInsufficientStock is a stable business result of Inventory, not a SQL error.
var ErrInsufficientStock = errors.New("insufficient stock for purchase return")
