package application

import (
	"context"
	"errors"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type paymentProjectionKey struct{}

type paymentProjectionTx struct {
	ctx        context.Context
	committed  bool
	rolledBack bool
}

func (t *paymentProjectionTx) Context() context.Context { return t.ctx }
func (t *paymentProjectionTx) Commit(context.Context) error {
	t.committed = true
	return nil
}
func (t *paymentProjectionTx) Rollback(context.Context) error {
	t.rolledBack = true
	return nil
}

type paymentProjectionWork struct{ tx *paymentProjectionTx }

func (w paymentProjectionWork) Begin(ctx context.Context, _ unitofwork.Options) (unitofwork.Transaction, error) {
	w.tx.ctx = context.WithValue(ctx, paymentProjectionKey{}, true)
	return w.tx, nil
}

type paymentProjectionRepo struct {
	PaymentRepository
	projectionErr error
	seenContext   context.Context
}

func (*paymentProjectionRepo) CurrentShift(context.Context, requestscope.Scope) (string, error) {
	return "shift-1", nil
}
func (*paymentProjectionRepo) LockPaymentOrder(context.Context, requestscope.Scope, domain.PaymentInput) (string, string, float64, bool, bool, bool, error) {
	return "ORDER-1", "listo", 20, false, false, false, nil
}
func (*paymentProjectionRepo) PaidForOrder(context.Context, requestscope.Scope, domain.PaymentInput) (float64, error) {
	return 0, nil
}
func (*paymentProjectionRepo) CreatePayment(context.Context, requestscope.Scope, domain.PaymentInput, string) (string, error) {
	return "payment-1", nil
}
func (r *paymentProjectionRepo) Payment(ctx context.Context, _ requestscope.Scope, id string) (domain.PaymentView, error) {
	r.seenContext = ctx
	if r.projectionErr != nil {
		return domain.PaymentView{}, r.projectionErr
	}
	return domain.PaymentView{ID: id, Amount: "10.00"}, nil
}

type paymentProjectionMethods struct{ ExpenseMethods }

func (paymentProjectionMethods) Find(context.Context, string, string, string) (domain.PaymentMethodView, error) {
	return domain.PaymentMethodView{Code: "card", SalesEnabled: true, Active: true}, nil
}

type paymentProjectionOrders struct{ PaymentOrders }

func (paymentProjectionOrders) CompleteDeliveredAccount(context.Context, requestscope.Scope, string) (bool, error) {
	return false, nil
}

func TestCreatePaymentReadsProjectionBeforeCommit(t *testing.T) {
	for _, tc := range []struct {
		name      string
		readError error
		committed bool
	}{
		{name: "success", committed: true},
		{name: "read_failure_rolls_back", readError: errors.New("projection unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &paymentProjectionTx{}
			repo := &paymentProjectionRepo{projectionErr: tc.readError}
			svc := NewPayments(repo, paymentProjectionWork{tx}, nil, paymentProjectionMethods{}, paymentProjectionOrders{}).WithAuthorization(collectionPermissions{allowed: map[string]bool{"cash.manage": true}})
			s := requestscope.Scope{OrganizationID: "org", LocationID: "loc", UserID: "user"}
			payment, err := svc.Create(context.Background(), s, domain.PaymentInput{
				OrderID: "order-1", Method: "card", Amount: 10,
			})
			if (err != nil) != (tc.readError != nil) {
				t.Fatalf("unexpected create error: %v", err)
			}
			if tx.committed != tc.committed {
				t.Fatalf("commit=%t; want %t", tx.committed, tc.committed)
			}
			if !tx.rolledBack {
				t.Fatal("transaction cleanup not invoked")
			}
			if repo.seenContext != tx.ctx || repo.seenContext.Value(paymentProjectionKey{}) != true {
				t.Fatal("payment projection must read within the active transaction")
			}
			if tc.readError == nil && (payment.ID != "payment-1" || payment.Amount != "10.00") {
				t.Fatalf("unexpected payment projection: %#v", payment)
			}
		})
	}
}

func (*paymentProjectionRepo) CanReadAllShifts(context.Context, scope) (bool, error) {
	return false, nil
}
