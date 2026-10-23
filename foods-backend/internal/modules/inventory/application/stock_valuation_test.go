package application

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
)

// valuationRepository keeps one balance and records what entries and adjustments
// write. Any other call panics.
type valuationRepository struct {
	StockRepository
	quantity, average float64
	settingsCalls     int
	itemExistsErr     error
	movements         []Movement
}

func (r *valuationRepository) LockEntryItem(context.Context, scope, domain.InventoryEntryInput) (string, string, string, string, *string, *string, error) {
	return "item-1", "ING-1", "Arroz", "kg", nil, nil, nil
}
func (r *valuationRepository) EntryLocationSettings(context.Context, scope, string, domain.InventoryEntryInput) (int64, error) {
	r.settingsCalls++
	return 1, nil
}
func (r *valuationRepository) CombinationAllowed(context.Context, scope, string, string) (bool, error) {
	return true, nil
}
func (r *valuationRepository) EnsurePresentation(context.Context, scope, string, string, float64) (string, error) {
	return "presentation-1", nil
}
func (r *valuationRepository) EnsureEntryBalance(context.Context, scope, string) (int64, error) {
	return 0, nil
}
func (r *valuationRepository) LockEntryBalance(context.Context, scope, string) (float64, float64, error) {
	return r.quantity, r.average, nil
}
func (r *valuationRepository) SaveBalance(_ context.Context, _ scope, _ string, quantity, average float64) (int64, error) {
	r.quantity, r.average = quantity, average
	return 1, nil
}
func (r *valuationRepository) CreateEntry(context.Context, scope, *string, string, domain.InventoryEntryInput, string) (string, error) {
	return "entry-1", nil
}
func (r *valuationRepository) LockAdjustmentItem(context.Context, scope, domain.InventoryAdjustmentInput) (*string, string, string, error) {
	return nil, "Arroz", "kg", nil
}
func (r *valuationRepository) EnsureAdjustmentBalance(context.Context, scope, domain.InventoryAdjustmentInput) (int64, error) {
	return 0, nil
}
func (r *valuationRepository) LockAdjustmentBalance(context.Context, scope, domain.InventoryAdjustmentInput) (float64, float64, error) {
	return r.quantity, r.average, nil
}
func (r *valuationRepository) UpdateAdjustmentBalance(_ context.Context, _ scope, _ domain.InventoryAdjustmentInput, quantity, average float64) (int64, error) {
	r.quantity, r.average = quantity, average
	return 1, nil
}
func (r *valuationRepository) CreateAdjustment(context.Context, scope, domain.InventoryAdjustmentInput, float64, float64) (string, time.Time, error) {
	return "adjustment-1", time.Time{}, nil
}
func (r *valuationRepository) SaveMovement(_ context.Context, _ scope, m Movement) error {
	r.movements = append(r.movements, m)
	return nil
}
func (r *valuationRepository) ItemExists(context.Context, scope, string) (bool, error) {
	return r.itemExistsErr == nil, r.itemExistsErr
}

func cost(value float64) *float64 { return &value }

func closeTo(a, b float64) bool { return math.Abs(a-b) < 0.00005 }

func TestEntryAdjustmentWithCostRecalculatesTheAverage(t *testing.T) {
	repo := &valuationRepository{quantity: 10, average: 2}
	stock := NewStock(repo, stockWork{}, noAudit{})
	_, err := stock.CreateAdjustment(context.Background(), scope{}, domain.InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "entry", Reason: "surplus_adjustment", Quantity: 10, UnitCost: cost(4)})
	if err != nil {
		t.Fatal(err)
	}
	m := repo.movements[0]
	if repo.quantity != 20 || !closeTo(repo.average, 3) || m.UnitCost != 4 || !closeTo(m.ValueDelta, 40) || !closeTo(m.BalanceValue, 60) {
		t.Fatalf("entry must value at its cost and balance at the new average: balance=%v avg=%v movement=%#v", repo.quantity, repo.average, m)
	}
}

func TestEntryAdjustmentWithoutCostUsesTheCurrentAverage(t *testing.T) {
	repo := &valuationRepository{quantity: 10, average: 2}
	stock := NewStock(repo, stockWork{}, noAudit{})
	if _, err := stock.CreateAdjustment(context.Background(), scope{}, domain.InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "entry", Reason: "surplus_adjustment", Quantity: 5}); err != nil {
		t.Fatal(err)
	}
	m := repo.movements[0]
	if repo.average != 2 || m.UnitCost != 2 || !closeTo(m.BalanceValue, 30) {
		t.Fatalf("entry without cost keeps the average: avg=%v movement=%#v", repo.average, m)
	}
}

func TestEntryWithoutAverageRequiresACost(t *testing.T) {
	for name, run := range map[string]func(*Stock) error{
		"adjustment": func(stock *Stock) error {
			_, err := stock.CreateAdjustment(context.Background(), scope{}, domain.InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "entry", Reason: "surplus_adjustment", Quantity: 5})
			return err
		},
		"entry": func(stock *Stock) error {
			_, err := stock.CreateEntry(context.Background(), scope{}, domain.InventoryEntryInput{InventoryItemID: "item-1", Unit: "kg", Quantity: 5})
			return err
		},
	} {
		repo := &valuationRepository{}
		if code := linkFailureCode(run(NewStock(repo, stockWork{}, noAudit{}))); code != "unit_cost_required" || len(repo.movements) != 0 {
			t.Fatalf("%s: a first entry without cost must be rejected, got %q", name, code)
		}
	}
}

func TestExitAdjustmentIgnoresCostAndKeepsTheAverage(t *testing.T) {
	repo := &valuationRepository{quantity: 10, average: 2}
	stock := NewStock(repo, stockWork{}, noAudit{})
	if _, err := stock.CreateAdjustment(context.Background(), scope{}, domain.InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "exit", Reason: "waste", Quantity: 4, UnitCost: cost(9)}); err != nil {
		t.Fatal(err)
	}
	m := repo.movements[0]
	if repo.quantity != 6 || repo.average != 2 || m.UnitCost != 2 || !closeTo(m.ValueDelta, -8) || !closeTo(m.BalanceValue, 12) {
		t.Fatalf("exit leaves at average cost: balance=%v avg=%v movement=%#v", repo.quantity, repo.average, m)
	}
}

func TestEntryKeepsTheMinimumUnlessItIsSent(t *testing.T) {
	repo := &valuationRepository{quantity: 1, average: 2}
	stock := NewStock(repo, stockWork{}, noAudit{})
	if _, err := stock.CreateEntry(context.Background(), scope{}, domain.InventoryEntryInput{InventoryItemID: "item-1", Unit: "kg", Quantity: 3, UnitCost: cost(5)}); err != nil {
		t.Fatal(err)
	}
	if repo.settingsCalls != 0 {
		t.Fatal("an entry without minimumStock must not touch the configured minimum")
	}
	m := repo.movements[0]
	if repo.quantity != 4 || !closeTo(repo.average, 4.25) || m.UnitCost != 5 || !closeTo(m.BalanceValue, 17) {
		t.Fatalf("entry must bring its cost into the average: balance=%v avg=%v movement=%#v", repo.quantity, repo.average, m)
	}
	if _, err := stock.CreateEntry(context.Background(), scope{}, domain.InventoryEntryInput{InventoryItemID: "item-1", Unit: "kg", Quantity: 1, MinimumStock: cost(2)}); err != nil {
		t.Fatal(err)
	}
	if repo.settingsCalls != 1 {
		t.Fatal("an explicit minimumStock must be saved")
	}
}

func TestStockCannotExceedTheColumnLimit(t *testing.T) {
	repo := &valuationRepository{quantity: domain.MaxStockQuantity - 1, average: 2}
	stock := NewStock(repo, stockWork{}, noAudit{})
	_, err := stock.CreateAdjustment(context.Background(), scope{}, domain.InventoryAdjustmentInput{InventoryItemID: "item-1", MovementType: "entry", Reason: "surplus_adjustment", Quantity: 5})
	if linkFailureCode(err) != "stock_limit_exceeded" {
		t.Fatalf("a balance past numeric(14,3) must be a validation error, got %v", err)
	}
}

func TestLocationSettingsSeparatesMissingItemsFromFailures(t *testing.T) {
	stock := NewStock(&valuationRepository{itemExistsErr: errors.New("connection reset")}, stockWork{}, noAudit{})
	if code := linkFailureCode(stock.LocationSettings(context.Background(), scope{}, "item-1", domain.LocationSettings{})); code != "inventory_unavailable" {
		t.Fatalf("a database failure is not a missing item: %q", code)
	}
	if code := linkFailureCode(stock.LocationSettings(context.Background(), scope{}, "item-1", domain.LocationSettings{MinimumStock: domain.MaxStockQuantity})); code != "invalid_inventory_settings" {
		t.Fatalf("levels past the column limit must be rejected: %q", code)
	}
}
