package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"testing"
)

type menuSnapshot struct{ OrderRepository }

// Not preloaded: the per-product lookups below keep validating it.
func (menuSnapshot) OrderProducts(context.Context, scope, []string) (map[string]domain.OrderProductSnapshot, error) {
	return map[string]domain.OrderProductSnapshot{}, nil
}
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

// preloadedMenu counts the per-product lookups that the preload must avoid.
type preloadedMenu struct {
	OrderRepository
	snapshots           map[string]domain.OrderProductSnapshot
	preloads, perItem   int
	modifierLookups     int
	requestedProductIDs []string
}

func (m *preloadedMenu) OrderProducts(_ context.Context, _ scope, ids []string) (map[string]domain.OrderProductSnapshot, error) {
	m.preloads++
	m.requestedProductIDs = ids
	return m.snapshots, nil
}
func (m *preloadedMenu) ProductIsCombo(context.Context, scope, *string) (bool, error) {
	m.perItem++
	return false, nil
}
func (m *preloadedMenu) ProductForOrder(context.Context, scope, *string) (string, float64, bool, error) {
	m.perItem++
	return "Chicha", 6, true, nil
}
func (m *preloadedMenu) ModifierGroups(context.Context, scope, string) (map[string]*domain.ModifierGroup, []string, error) {
	m.modifierLookups++
	return map[string]*domain.ModifierGroup{}, nil, nil
}

func TestPreparationPreloadsProductsInOneLookup(t *testing.T) {
	m := &preloadedMenu{snapshots: map[string]domain.OrderProductSnapshot{
		"lomo":    {Name: "Lomo saltado", Price: 30, Available: true},
		"ceviche": {Name: "Ceviche", Price: 35, Available: true},
	}}
	service := New(m, nil, nil, nil)
	items, total, err := service.prepareItems(context.Background(), scope{}, "", []domain.OrderItemInput{
		{ProductID: " lomo ", Qty: 2}, {ProductID: "ceviche", Qty: 1}, {ProductID: "chicha", Qty: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.preloads != 1 || len(m.requestedProductIDs) != 3 || m.requestedProductIDs[0] != "lomo" {
		t.Fatalf("expected one trimmed preload for every line, got %d %v", m.preloads, m.requestedProductIDs)
	}
	// Only "chicha" was not preloaded: it falls back to its own two lookups.
	if m.perItem != 2 || m.modifierLookups != 1 {
		t.Fatalf("preloaded products without modifiers must skip lookups: perItem=%d modifiers=%d", m.perItem, m.modifierLookups)
	}
	if total != 101 || len(items) != 3 || items[0].Name != "Lomo saltado" || items[0].UnitPrice != 30 || items[0].Modifiers == nil {
		t.Fatalf("unexpected items: total=%v %+v", total, items)
	}
}

func TestPreparationStillValidatesPreloadedModifiersAndAvailability(t *testing.T) {
	m := &preloadedMenu{snapshots: map[string]domain.OrderProductSnapshot{
		"cafe":   {Name: "Café", Price: 8, Available: true, HasModifiers: true},
		"agua":   {Name: "Agua", Price: 3, Available: true},
		"postre": {Name: "Postre", Price: 9, Available: false},
	}}
	service := New(m, nil, nil, nil)
	if _, _, err := service.prepareItems(context.Background(), scope{}, "", []domain.OrderItemInput{{ProductID: "cafe", Qty: 1}}); err != nil || m.modifierLookups != 1 {
		t.Fatalf("product with modifier groups must load them: err=%v lookups=%d", err, m.modifierLookups)
	}
	_, _, err := service.prepareItems(context.Background(), scope{}, "", []domain.OrderItemInput{{ProductID: "agua", Qty: 1, Modifiers: []domain.OrderItemModifierInput{{GroupID: "g", OptionID: "o"}}}})
	var failure *apperror.Error
	if !errors.As(err, &failure) || failure.Code != "invalid_modifier_selection" {
		t.Fatalf("foreign modifiers must still be rejected, got %v", err)
	}
	_, _, err = service.prepareItems(context.Background(), scope{}, "", []domain.OrderItemInput{{ProductID: "postre", Qty: 1}})
	if !errors.As(err, &failure) || failure.Code != "product_unavailable" {
		t.Fatalf("unavailable preloaded product accepted: %v", err)
	}
}
