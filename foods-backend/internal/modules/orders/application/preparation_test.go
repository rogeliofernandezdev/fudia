package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"testing"
)

type menuSnapshot struct{ OrderRepository }

func (menuSnapshot) ProductIsCombo(context.Context, scope, *string) (bool, error) { return false, nil }
func (menuSnapshot) ProductForOrder(context.Context, scope, *string) (string, float64, bool, error) {
	return "Café", 12, true, nil
}
func (menuSnapshot) ModifierGroups(context.Context, scope, string) (map[string]*domain.ModifierGroup, []string, error) {
	return map[string]*domain.ModifierGroup{"milk": {ID: "milk", Name: "Leche", Min: 1, Max: 1, Required: true, Options: map[string]domain.ModifierOption{"oat": {ID: "oat", Name: "Avena", Surcharge: 3}}}}, []string{"milk"}, nil
}
func TestPreparationUsesCatalogPriceAndModifierSurcharge(t *testing.T) {
	service := New(menuSnapshot{}, nil, nil, nil)
	items, total, err := service.prepareItems(context.Background(), scope{}, "", []domain.OrderItemInput{{ProductID: "coffee", Qty: 2, UnitPrice: 0.01, Modifiers: []domain.OrderItemModifierInput{{GroupID: "milk", OptionID: "oat"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if total != 30 || len(items) != 1 || items[0].UnitPrice != 15 || items[0].Name != "Café" {
		t.Fatalf("client price replaced authoritative snapshot: total=%v items=%+v", total, items)
	}
}
func TestPreparationRejectsMissingRequiredModifier(t *testing.T) {
	service := New(menuSnapshot{}, nil, nil, nil)
	_, _, err := service.prepareItems(context.Background(), scope{}, "", []domain.OrderItemInput{{ProductID: "coffee", Qty: 1}})
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != "invalid_modifier_selection" {
		t.Fatalf("expected modifier selection error, got %v", err)
	}
}
