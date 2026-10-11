package application

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/contracts"
)

type purchaseLedgerFake struct {
	balance contracts.Balance
	current, average float64
	updatedQuantity, updatedCost float64
	movement contracts.Movement
	writes int
	seenContext context.Context
	movementErr error
}
func (f *purchaseLedgerFake) EnsureEntryBalance(ctx context.Context, _ scope, _ string) (int64, error) {
	f.seenContext = ctx
	return 1, nil
}
func (f *purchaseLedgerFake) LockEntryBalance(ctx context.Context, _ scope, _ string) (float64, float64, error) {
	f.seenContext = ctx
	return f.current, f.average, nil
}
func (f *purchaseLedgerFake) LockBalance(ctx context.Context, _ scope, _ string) (contracts.Balance, error) {
	f.seenContext = ctx
	return f.balance, nil
}
func (f *purchaseLedgerFake) SaveBalance(ctx context.Context, _ scope, _ string, quantity, cost float64) (int64, error) {
	f.seenContext = ctx
	f.updatedQuantity, f.updatedCost = quantity, cost
	f.writes++
	return 1, nil
}
func (f *purchaseLedgerFake) SaveMovement(ctx context.Context, _ scope, m contracts.Movement) error {
	f.seenContext = ctx
	f.movement = m
	f.writes++
	return f.movementErr
}
func near(a, b float64) bool { return math.Abs(a-b) < 0.00001 }

func TestPurchaseLedgerOwnsWeightedAverageAndKardex(t *testing.T) {
	ctx := context.WithValue(context.Background(), "transaction", "active")
	f := &purchaseLedgerFake{current: 10, average: 2}
	ledger := NewPurchaseLedger(f)
	err := ledger.ReceivePurchase(ctx, scope{}, contracts.PurchaseReceipt{
		ItemID: "item", Quantity: 5, UnitsPerPresentation: 10,
		UnitCost: 30, ReceiptID: "receipt", Note: "recepcion",
	})
	if err != nil { t.Fatal(err) }
	if f.seenContext != ctx { t.Fatal("caller transaction context not forwarded") }
	if !near(f.updatedQuantity, 60) || !near(f.updatedCost, 2.8333) {
		t.Fatalf("wrong balance or weighted average: %v %v", f.updatedQuantity, f.updatedCost)
	}
	if f.writes != 2 { t.Fatalf("want balance + movement writes; got %d", f.writes) }
	if f.movement.Type != "entry" || f.movement.SourceType != "purchase_receipt" ||
		!near(f.movement.Delta, 50) || !near(f.movement.UnitCost, 3) ||
		!near(f.movement.ValueDelta, 150) || !near(f.movement.BalanceValue, 169.998) {
		t.Fatalf("wrong purchase ledger movement: %#v", f.movement)
	}
}

func TestPurchaseReturnPreservesAverageAndRejectsShortage(t *testing.T) {
	f := &purchaseLedgerFake{balance: contracts.Balance{Quantity: 8, AverageUnitCost: 3}}
	ledger := NewPurchaseLedger(f)
	ctx := context.Background()
	err := ledger.ReturnPurchase(ctx, scope{}, contracts.PurchaseReturn{
		ItemID: "item", Quantity: 2, UnitsPerPresentation: 3,
		Kind: "supplier_return", ReturnID: "return-1",
	})
	if err != nil { t.Fatal(err) }
	if !near(f.updatedQuantity, 2) || !near(f.updatedCost, 3) ||
		!near(f.movement.Delta, -6) || !near(f.movement.ValueDelta, -18) ||
		!near(f.movement.BalanceValue, 6) {
		t.Fatalf("wrong return balance or movement: %#v, balance %f/%f", f.movement, f.updatedQuantity, f.updatedCost)
	}
	f.writes = 0
	err = ledger.ReturnPurchase(ctx, scope{}, contracts.PurchaseReturn{
		ItemID: "item", Quantity: 3, UnitsPerPresentation: 3,
		Kind: "receipt_correction", ReturnID: "return-2",
	})
	if !errors.Is(err, contracts.ErrInsufficientStock) || f.writes != 0 {
		t.Fatalf("shortage must reject without writes: err=%v writes=%d", err, f.writes)
	}
}

func TestPurchaseLedgerPropagatesMovementFailuresForCallerRollback(t *testing.T) {
	f := &purchaseLedgerFake{movementErr: errors.New("movement failed")}
	ledger := NewPurchaseLedger(f)
	err := ledger.ReceivePurchase(context.Background(), scope{}, contracts.PurchaseReceipt{
		ItemID: "item", Quantity: 1, UnitsPerPresentation: 1, UnitCost: 2, ReceiptID: "receipt",
	})
	if !errors.Is(err, f.movementErr) {
		t.Fatalf("caller must receive persistence error for transaction rollback: %v", err)
	}
}
