package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// linkRepository implements only what linking uses; any other call panics.
type linkRepository struct {
	StockRepository
	candidate domain.LinkProductCandidate
	lockErr   error
	open      bool
	itemUnit  string
	marked    bool
	linked    bool
}

func (r *linkRepository) UnitExists(context.Context, scope, string) (bool, error) { return true, nil }
func (r *linkRepository) LockLinkableProduct(context.Context, scope, string) (domain.LinkProductCandidate, error) {
	return r.candidate, r.lockErr
}
func (r *linkRepository) ProductInOpenOrders(context.Context, scope, string) (bool, error) {
	return r.open, nil
}
func (r *linkRepository) MarkProductInventoryControlled(context.Context, scope, domain.LinkProductCandidate) error {
	r.marked = true
	return nil
}
func (r *linkRepository) LinkEntryProduct(_ context.Context, _ scope, in domain.InventoryEntryInput, _ string, _ string) (string, string, error) {
	r.linked = true
	unit := r.itemUnit
	if unit == "" {
		unit = in.Unit
	}
	return "item-1", unit, nil
}

func linkFailureCode(err error) string {
	var e *apperror.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestLinkExistingProductKeepsTheCommercialRecord(t *testing.T) {
	repo := &linkRepository{candidate: domain.LinkProductCandidate{ID: "product-1", SKU: "PROD-0001", Name: "Inca Kola 500 ml", QuantityControl: "portions"}}
	stock := &Stock{repo: repo}
	out, err := stock.linkExistingProduct(context.Background(), scope{}, domain.InventoryEntryInput{ProductID: "product-1", Unit: "botella"})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.marked || !repo.linked || out.ProductID == nil || *out.ProductID != "product-1" || out.InventoryItemID != "item-1" || out.Name != "Inca Kola 500 ml" || out.Kind != "product" {
		t.Fatalf("product must be reused and linked: %#v %#v", out, repo)
	}
}

func TestLinkExistingProductRejectsUnsafeConversions(t *testing.T) {
	for _, tc := range []struct {
		name string
		repo *linkRepository
		code string
	}{
		{"missing", &linkRepository{lockErr: ErrNotFound}, "product_not_found"},
		{"combo", &linkRepository{candidate: domain.LinkProductCandidate{ID: "p", QuantityControl: "none", IsCombo: true}}, "product_link_combo"},
		{"recipe", &linkRepository{candidate: domain.LinkProductCandidate{ID: "p", QuantityControl: "portions", HasRecipe: true}}, "product_link_recipe"},
		{"open orders", &linkRepository{candidate: domain.LinkProductCandidate{ID: "p", QuantityControl: "portions"}, open: true}, "quantity_control_open_orders"},
		{"unit", &linkRepository{candidate: domain.LinkProductCandidate{ID: "p", QuantityControl: "inventory"}, itemUnit: "lata"}, "inventory_unit_conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (&Stock{repo: tc.repo}).linkExistingProduct(context.Background(), scope{}, domain.InventoryEntryInput{ProductID: "p", Unit: "botella"})
			if code := linkFailureCode(err); code != tc.code {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
			if tc.code != "inventory_unit_conflict" && tc.repo.marked {
				t.Fatal("rejected conversion must not change the product")
			}
		})
	}
}
