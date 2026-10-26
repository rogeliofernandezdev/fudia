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
	movements []Movement
	locks     [][]string
	missing   bool
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
func (r *orderStockRepository) LockProductInventory(context.Context, scope, string) (string, error) {
	return "item-1", nil
}
func (r *orderStockRepository) EnsureEntryBalance(context.Context, scope, string) (int64, error) {
	return 0, nil
}
func (r *orderStockRepository) LockEntryBalance(context.Context, scope, string) (float64, float64, error) {
	return 10, 2.5, nil
}
func (r *orderStockRepository) SaveBalance(context.Context, scope, string, float64, float64) (int64, error) {
	return 1, nil
}
func (r *orderStockRepository) SaveMovement(_ context.Context, _ scope, m Movement) error {
	r.movements = append(r.movements, m)
	return nil
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
