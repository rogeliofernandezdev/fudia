package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/orders/domain"
)

// paymentOrderRepo returns one order with its lines and per-line preparation state.
type paymentOrderRepo struct {
	OrderRepository
	serviceErr error
}

func (paymentOrderRepo) Record(_ context.Context, _ scope, id string) (domain.Order, error) {
	return domain.Order{ID: id, Code: "PED-1", Status: "confirmado"}, nil
}
func (paymentOrderRepo) Items(context.Context, scope, string, string) ([]domain.OrderItem, error) {
	return []domain.OrderItem{{ID: "line-ready"}, {ID: "line-pending"}}, nil
}
func (r paymentOrderRepo) ServiceItems(context.Context, scope, string) ([]domain.OrderServiceItem, error) {
	if r.serviceErr != nil {
		return nil, r.serviceErr
	}
	return []domain.OrderServiceItem{
		{ID: "s1", OrderItemID: "line-ready", Status: "listo", StatusLabel: "Por entregar"},
		{ID: "s2", OrderItemID: "line-pending", Status: "confirmado", StatusLabel: "Pendiente"},
	}, nil
}

// Cobro y Venta leen PaymentOrder: deben recibir el estado real de cada línea,
// no inferirlo del estado general del pedido.
func TestPaymentOrderIncludesPreparationStatePerLine(t *testing.T) {
	a := &Orders{repo: paymentOrderRepo{}}
	o, err := a.PaymentOrder(context.Background(), scope{}, "order")
	if err != nil {
		t.Fatalf("PaymentOrder: %v", err)
	}
	if len(o.Items) != 2 || len(o.ServiceItems) != 2 {
		t.Fatalf("items=%d serviceItems=%d, want 2 and 2", len(o.Items), len(o.ServiceItems))
	}
	states := map[string]string{}
	for _, item := range o.ServiceItems {
		states[item.OrderItemID] = item.Status
	}
	if states["line-ready"] != "listo" || states["line-pending"] != "confirmado" {
		t.Fatalf("states=%v, want each line with its own preparation state", states)
	}
}

func TestPaymentOrderFailsExplicitlyWhenPreparationStateIsUnavailable(t *testing.T) {
	a := &Orders{repo: paymentOrderRepo{serviceErr: errors.New("db down")}}
	if _, err := a.PaymentOrder(context.Background(), scope{}, "order"); err == nil {
		t.Fatal("PaymentOrder must not hide a failed preparation-state read")
	}
}
