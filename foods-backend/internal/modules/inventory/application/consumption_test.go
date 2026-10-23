package application

import (
	"context"
	"strings"
	"testing"
)

// orderStockRepository covers an inventory-controlled product sold by an order.
type orderStockRepository struct {
	StockRepository
	movements []Movement
}

func (r *orderStockRepository) OrderBusinessDate(context.Context, scope, string) (string, error) {
	return "2026-10-23", nil
}
func (r *orderStockRepository) LockQuantityProduct(context.Context, scope, string) (string, string, error) {
	return "Inca Kola 500 ml", "inventory", nil
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
