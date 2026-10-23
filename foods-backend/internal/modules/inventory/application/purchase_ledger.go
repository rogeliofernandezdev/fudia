package application

import (
	"context"
	"errors"
	"math"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/contracts"
)

// PurchaseLedgerRepository exposes only the persistence primitives used by
// Inventory's purchase ledger use cases. The PostgreSQL adapter owns all SQL.
type PurchaseLedgerRepository interface {
	EnsureEntryBalance(context.Context, scope, string) (int64, error)
	LockEntryBalance(context.Context, scope, string) (float64, float64, error)
	LockBalance(context.Context, scope, string) (contracts.Balance, error)
	SaveBalance(context.Context, scope, string, float64, float64) (int64, error)
	SaveMovement(context.Context, scope, contracts.Movement) error
}

// PurchaseLedger executes within a caller-supplied transaction context.
// It must not begin or commit a nested transaction.
type PurchaseLedger struct{ repo PurchaseLedgerRepository }

func NewPurchaseLedger(repo PurchaseLedgerRepository) *PurchaseLedger {
	return &PurchaseLedger{repo: repo}
}

func (l *PurchaseLedger) ReceivePurchase(ctx context.Context, s scope, in contracts.PurchaseReceipt) error {
	if in.ItemID == "" || in.ReceiptID == "" || in.Quantity <= 0 || in.UnitsPerPresentation <= 0 ||
		math.IsNaN(in.Quantity) || math.IsNaN(in.UnitsPerPresentation) || math.IsNaN(in.UnitCost) ||
		math.IsInf(in.Quantity, 0) || math.IsInf(in.UnitsPerPresentation, 0) || math.IsInf(in.UnitCost, 0) {
		return errors.New("invalid inventory receipt")
	}
	quantity := math.Round(in.Quantity*in.UnitsPerPresentation*1000) / 1000
	if _, err := l.repo.EnsureEntryBalance(ctx, s, in.ItemID); err != nil {
		return err
	}
	current, currentAverage, err := l.repo.LockEntryBalance(ctx, s, in.ItemID)
	if err != nil {
		return err
	}
	balance := math.Round((current+quantity)*1000) / 1000
	unitCost := in.UnitCost / in.UnitsPerPresentation
	average := currentAverage
	if balance > 0 {
		average = math.Round(((current*currentAverage)+(quantity*unitCost))/balance*10000) / 10000
	}
	if _, err = l.repo.SaveBalance(ctx, s, in.ItemID, balance, average); err != nil {
		return err
	}
	return l.repo.SaveMovement(ctx, s, contracts.Movement{
		ProductID: in.ProductID, ItemID: in.ItemID, Type: "entry",
		Delta: quantity, Balance: balance, UnitCost: unitCost,
		SourceType: "purchase_receipt", SourceID: in.ReceiptID, Note: in.Note,
		ValueDelta: math.Round(quantity*unitCost*10000) / 10000,
		BalanceValue: math.Round(balance*average*10000) / 10000,
	})
}

func (l *PurchaseLedger) ReturnPurchase(ctx context.Context, s scope, in contracts.PurchaseReturn) error {
	if in.ItemID == "" || in.ReturnID == "" || in.Quantity <= 0 || in.UnitsPerPresentation <= 0 || in.UnitCost < 0 ||
		(in.Kind != "supplier_return" && in.Kind != "receipt_correction") ||
		math.IsNaN(in.Quantity) || math.IsNaN(in.UnitsPerPresentation) || math.IsNaN(in.UnitCost) ||
		math.IsInf(in.Quantity, 0) || math.IsInf(in.UnitsPerPresentation, 0) || math.IsInf(in.UnitCost, 0) {
		return errors.New("invalid inventory return")
	}
	quantity := math.Round(in.Quantity*in.UnitsPerPresentation*1000) / 1000
	balance, err := l.repo.LockBalance(ctx, s, in.ItemID)
	if err != nil {
		return err
	}
	if quantity > balance.Quantity+0.000001 {
		return contracts.ErrInsufficientStock
	}
	// The outflow reverses the receipt at its own cost, so a wrong receipt does not
	// leave its price inside the average. Consumption at average cost can leave less
	// value than the receipt carried; the remaining value never goes below zero.
	unitCost := in.UnitCost / in.UnitsPerPresentation
	after := math.Round((balance.Quantity-quantity)*1000) / 1000
	valueBefore := balance.Quantity * balance.AverageUnitCost
	average := balance.AverageUnitCost
	if after > 0 {
		average = math.Round(math.Max(valueBefore-quantity*unitCost, 0)/after*10000) / 10000
	}
	balanceValue := math.Round(after*average*10000) / 10000
	if _, err = l.repo.SaveBalance(ctx, s, in.ItemID, after, average); err != nil {
		return err
	}
	return l.repo.SaveMovement(ctx, s, contracts.Movement{
		ProductID: balance.ProductID, ItemID: in.ItemID, Type: in.Kind,
		Delta: -quantity, Balance: after, UnitCost: math.Round(unitCost*10000) / 10000,
		SourceType: "purchase_return", SourceID: in.ReturnID, Note: in.Note,
		ValueDelta:   math.Round((balanceValue-valueBefore)*10000) / 10000,
		BalanceValue: balanceValue,
	})
}
