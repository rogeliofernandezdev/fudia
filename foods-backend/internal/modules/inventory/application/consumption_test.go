package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"reflect"
	"strings"
	"testing"
)

// orderStockRepository covers an inventory-controlled product sold by an order.
type orderStockRepository struct {
	StockRepository
	movements       []Movement
	balances        []BalanceWrite
	locks           [][]string
	balanceLocks    [][]string
	ingredientReads [][]string
	writes          int
	missing         bool
	inactive        string
}

func (r *orderStockRepository) OrderBusinessDate(context.Context, scope, string) (string, error) {
	return "2026-10-23", nil
}
func (r *orderStockRepository) LockQuantityProducts(_ context.Context, _ scope, ids []string) (map[string]domain.QuantityProduct, error) {
	r.locks = append(r.locks, ids)
	out := map[string]domain.QuantityProduct{}
	for _, id := range ids {
		if r.missing && id == "product-2" {
			continue
		}
		out[id] = domain.QuantityProduct{Name: "Inca Kola 500 ml", Control: "inventory"}
	}
	return out, nil
}
func (r *orderStockRepository) LockPortionsFor(context.Context, scope, []string, string) (map[string]domain.PortionLock, error) {
	return map[string]domain.PortionLock{}, nil
}
func (r *orderStockRepository) LockProductInventories(_ context.Context, _ scope, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		out[id] = strings.Replace(id, "product", "item", 1)
	}
	return out, nil
}
func (r *orderStockRepository) LockBalances(_ context.Context, _ scope, ids []string) (map[string]domain.InventoryBalanceLock, error) {
	r.balanceLocks = append(r.balanceLocks, ids)
	out := map[string]domain.InventoryBalanceLock{}
	for _, id := range ids {
		if id == r.inactive {
			continue
		}
		out[id] = domain.InventoryBalanceLock{Name: "Insumo " + id, Quantity: 10, AverageUnitCost: 2.5}
	}
	return out, nil
}
func (r *orderStockRepository) ApplyStockWrites(_ context.Context, _ scope, w StockWrites) error {
	r.writes++
	r.balances = append(r.balances, w.Balances...)
	r.movements = append(r.movements, w.Movements...)
	return nil
}
func (r *orderStockRepository) RecipeModules(context.Context, scope) (bool, bool, error) {
	return true, true, nil
}
func (r *orderStockRepository) ActiveRecipes(_ context.Context, _ scope, ids []string) (map[string]domain.ActiveRecipe, error) {
	out := map[string]domain.ActiveRecipe{}
	for _, id := range ids {
		out[id] = domain.ActiveRecipe{ID: "recipe-" + id, Yield: 1}
	}
	return out, nil
}
func (r *orderStockRepository) RecipeIngredientsFor(_ context.Context, _ scope, ids []string) (map[string][]domain.RecipeIngredient, error) {
	r.ingredientReads = append(r.ingredientReads, ids)
	out := map[string][]domain.RecipeIngredient{}
	for _, id := range ids {
		out[id] = []domain.RecipeIngredient{{ItemID: "item-rice", Quantity: 0.2}, {ItemID: "item-" + id, Quantity: 1, WastePercent: 10}}
	}
	return out, nil
}

func TestOrderMovementNoteDoesNotExposeTheOrderID(t *testing.T) {
	const orderID = "33333333-3333-4333-8333-333333333333"
	cases := map[string]struct {
		change float64
		note   string
	}{
		"sale":            {2, "Salida automática por venta"},
		"sale_reversal":   {-2, "Reversa de venta"},
		"sale_adjustment": {1, "Ajuste por edición del pedido"},
	}
	for movementType, want := range cases {
		repo := &orderStockRepository{}
		stock := &Stock{repo: repo}
		if err := stock.ApplyOrderQuantityDelta(context.Background(), scope{}, orderID, map[string]float64{"product-1": want.change}, movementType); err != nil {
			t.Fatalf("%s: %v", movementType, err)
		}
		if len(repo.movements) != 1 {
			t.Fatalf("%s: expected one movement, got %d", movementType, len(repo.movements))
		}
		m := repo.movements[0]
		if m.Note != want.note || strings.Contains(m.Note, orderID) || m.SourceID != orderID {
			t.Fatalf("%s: unexpected movement %#v", movementType, m)
		}
	}
}

func TestOrderQuantityLocksAllProductsOnceInIDOrder(t *testing.T) {
	repo := &orderStockRepository{}
	stock := &Stock{repo: repo}
	delta := map[string]float64{"product-3": 1, "product-1": 2, "product-2": 1}
	if err := stock.ApplyOrderQuantityDelta(context.Background(), scope{}, "order-1", delta, "sale"); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"product-1", "product-2", "product-3"}}; !reflect.DeepEqual(repo.locks, want) {
		t.Fatalf("expected a single sorted lock, got %v", repo.locks)
	}
	if len(repo.movements) != 3 {
		t.Fatalf("every product must still move stock, got %d", len(repo.movements))
	}
	missing := &orderStockRepository{missing: true}
	err := (&Stock{repo: missing}).ApplyOrderQuantityDelta(context.Background(), scope{}, "order-1", delta, "sale")
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != "product_unavailable" {
		t.Fatalf("missing product must be reported, got %v", err)
	}
}

func TestOrderQuantityLocksBalancesOnceAndWritesInOneBatch(t *testing.T) {
	repo := &orderStockRepository{}
	stock := &Stock{repo: repo}
	delta := map[string]float64{"product-3": 1, "product-1": 2, "product-2": 1}
	if err := stock.ApplyOrderQuantityDelta(context.Background(), scope{}, "order-1", delta, "sale"); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"item-1", "item-2", "item-3"}}; !reflect.DeepEqual(repo.balanceLocks, want) {
		t.Fatalf("balances must be locked once in id order, got %v", repo.balanceLocks)
	}
	if repo.writes != 1 || len(repo.balances) != 3 || repo.balances[0].ItemID != "item-1" || repo.balances[0].Quantity != 8 {
		t.Fatalf("expected one batch with every balance, got %d writes %#v", repo.writes, repo.balances)
	}
}

func TestRecipeUsageReadsIngredientsOnceAndKeepsMessages(t *testing.T) {
	repo := &orderStockRepository{}
	stock := &Stock{repo: repo}
	usage, err := stock.DesiredRecipeUsage(context.Background(), scope{}, map[string]float64{"b": 2, "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"recipe-a", "recipe-b"}}; !reflect.DeepEqual(repo.ingredientReads, want) {
		t.Fatalf("ingredients must be read in one query, got %v", repo.ingredientReads)
	}
	if usage["item-rice"] != 0.6 || usage["item-recipe-a"] != 1.1 || usage["item-recipe-b"] != 2.2 {
		t.Fatalf("unexpected usage %#v", usage)
	}
	if err = stock.ApplyRecipeUsageDelta(context.Background(), scope{}, "order-1", map[string]float64{"item-z": 1, "item-a": -2, "item-zero": 0}); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"item-a", "item-z"}}; !reflect.DeepEqual(repo.balanceLocks, want) {
		t.Fatalf("recipe balances must be locked once in id order without zero changes, got %v", repo.balanceLocks)
	}
	if repo.writes != 1 || len(repo.movements) != 2 || repo.movements[0].Type != "recipe_reversal" || repo.movements[1].Type != "recipe_consumption" || repo.movements[1].Balance != 9 {
		t.Fatalf("unexpected recipe writes %d %#v", repo.writes, repo.movements)
	}
	var failure *apperror.Error
	err = stock.ApplyRecipeUsageDelta(context.Background(), scope{}, "order-1", map[string]float64{"item-a": 11})
	if !errors.As(err, &failure) || failure.Code != "insufficient_recipe_stock" || failure.Message != "No hay stock suficiente de Insumo item-a para preparar el pedido." {
		t.Fatalf("insufficient stock must keep its message, got %v", err)
	}
	err = (&Stock{repo: &orderStockRepository{inactive: "item-a"}}).ApplyRecipeUsageDelta(context.Background(), scope{}, "order-1", map[string]float64{"item-a": 1})
	if !errors.As(err, &failure) || failure.Code != "recipe_item_unavailable" {
		t.Fatalf("inactive item must be reported, got %v", err)
	}
}
