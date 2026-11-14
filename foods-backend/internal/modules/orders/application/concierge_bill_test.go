package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

type conciergeBillRepo struct {
	OrderRepository
	order       domain.Order
	pending     bool
	deliveryErr error
}

func (r *conciergeBillRepo) OpenTableOrder(context.Context, scope, string) (domain.Order, error) {
	return r.order, nil
}
func (r *conciergeBillRepo) UndeliveredItems(context.Context, scope, string) (bool, error) {
	return r.pending, r.deliveryErr
}
func (r *conciergeBillRepo) Items(context.Context, scope, string, string) ([]domain.OrderItem, error) {
	return []domain.OrderItem{{Name: "Lomo saltado", Qty: "1", UnitPrice: "25.00"}}, nil
}
func (r *conciergeBillRepo) NetPaid(context.Context, scope, string) (float64, error) { return 5, nil }
func (r *conciergeBillRepo) ConciergeOrderActive(context.Context, scope, string, string) (bool, error) {
	return r.order.CompletedAt == "", nil
}
func (r *conciergeBillRepo) Record(context.Context, scope, string) (domain.Order, error) {
	return r.order, nil
}

func TestConciergeBillRequiresEveryProductDelivered(t *testing.T) {
	for _, scenario := range []struct {
		name, status, completed string
		pending, allowed        bool
	}{
		{"saved", "nuevo", "", true, false},
		{"confirmed", "confirmado", "", true, false},
		{"preparing", "preparando", "", true, false},
		{"ready_is_not_delivered", "listo", "", true, false},
		{"partial_delivery", "listo", "", true, false},
		{"stale_aggregate_with_new_round", "entregado", "", true, false},
		{"all_delivered", "entregado", "", false, true},
		{"closed", "entregado", "2026-11-14", false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repo := &conciergeBillRepo{order: domain.Order{ID: "order", TableID: "table", Status: scenario.status, CompletedAt: scenario.completed, Total: "25.00"}, pending: scenario.pending}
			app := New(repo, nil, nil, nil)
			bill, err := app.TableBill(context.Background(), scope{}, "table")
			if scenario.allowed {
				if err != nil || bill.Total != "25.00" || bill.RemainingAmount != "20.00" {
					t.Fatalf("bill=%+v err=%v", bill, err)
				}
			} else {
				var failure *apperror.Error
				if !errors.As(err, &failure) || failure.Code != "products_not_delivered" {
					t.Fatalf("expected delivery conflict, got %v", err)
				}
			}
			state, err := app.ConciergeOrder(context.Background(), scope{}, "conversation", "order")
			if err != nil || state.CanRequestBill != scenario.allowed {
				t.Fatalf("availability=%v want=%v err=%v", state.CanRequestBill, scenario.allowed, err)
			}
		})
	}
}

func TestConciergeBillDoesNotAllowUnknownDeliveryOrOrdersWithoutTable(t *testing.T) {
	repo := &conciergeBillRepo{order: domain.Order{ID: "order", TableID: "table", Status: "entregado"}, deliveryErr: errors.New("database unavailable")}
	app := New(repo, nil, nil, nil)
	if _, err := app.TableBill(context.Background(), scope{}, "table"); err == nil {
		t.Fatal("delivery failure must not permit a bill")
	}
	repo.order.TableID = ""
	available, err := app.canRequestTableBill(context.Background(), scope{}, repo.order)
	if available || err != nil {
		t.Fatalf("non-table availability=%v err=%v", available, err)
	}
}
