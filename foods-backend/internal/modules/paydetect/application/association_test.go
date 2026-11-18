package application

import (
	"context"
	"testing"
	"time"

	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
)

type associationRepo struct {
	AdminRepository
	notification domain.Notification
}

func (r *associationRepo) Notification(context.Context, scope, string) (domain.Notification, error) {
	return r.notification, nil
}

func (r *associationRepo) LinkNotification(_ context.Context, _ scope, _ string, orderID, paymentID, _ string, _ time.Time) (domain.Notification, error) {
	r.notification.Status = domain.NotificationAssociated
	r.notification.OrderID, r.notification.PaymentID = orderID, paymentID
	return r.notification, nil
}

type associationOrders struct{}

func (associationOrders) PayOrderRef(_ context.Context, _ scope, id string) (ordercontracts.PayOrderRef, error) {
	return ordercontracts.PayOrderRef{ID: id}, nil
}

type associationRegisterer struct {
	inputs []billingcontracts.PayRegisterInput
}

func (r *associationRegisterer) PayRegister(_ context.Context, _ scope, in billingcontracts.PayRegisterInput) (billingcontracts.PayRegisterResult, error) {
	r.inputs = append(r.inputs, in)
	return billingcontracts.PayRegisterResult{PaymentID: "payment"}, nil
}

func TestAssociationKeepsDetectedProviderAndBankReference(t *testing.T) {
	for _, provider := range []string{domain.ProviderYape, domain.ProviderPlin} {
		for _, operation := range []string{"", "0012345678"} {
			repo := &associationRepo{notification: domain.Notification{Provider: provider, Status: domain.NotificationDetected, Amount: "10.50", EventKey: "opaque-android-event-key", OperationNumber: operation}}
			registerer := &associationRegisterer{}
			service := NewNotifications(repo).WithOrderLooker(associationOrders{}).WithPaymentRegisterer(registerer)
			for _, method := range []string{"cash", "card", "wallet", map[string]string{"yape": "plin", "plin": "yape"}[provider]} {
				if _, err := service.Associate(context.Background(), scope{}, "notification", "order", method, "cashier", time.Now()); err == nil {
					t.Fatalf("provider %s accepted method %s", provider, method)
				}
				if len(registerer.inputs) != 0 {
					t.Fatal("a rejected method registered money")
				}
			}
			out, err := service.Associate(context.Background(), scope{}, "notification", "order", provider, "cashier", time.Now())
			if err != nil || out.Status != domain.NotificationAssociated || len(registerer.inputs) != 1 {
				t.Fatalf("association failed: %+v %v", out, err)
			}
			in := registerer.inputs[0]
			wantReference := "Fudia Pay · " + map[string]string{"yape": "Yape", "plin": "Plin"}[provider]
			if operation != "" {
				wantReference += " · Operación " + operation
			}
			if in.Method != provider || in.Reference != wantReference || in.Amount != 10.5 {
				t.Fatalf("wrong payment: %+v", in)
			}
		}
	}
}
