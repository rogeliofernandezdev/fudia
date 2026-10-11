package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"testing"
)

type methodTx struct {
	ctx                   context.Context
	committed, rolledBack bool
}

func (t *methodTx) Context() context.Context       { return t.ctx }
func (t *methodTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *methodTx) Rollback(context.Context) error { t.rolledBack = true; return nil }

type methodWork struct{ tx *methodTx }

func (w methodWork) Begin(context.Context, unitofwork.Options) (unitofwork.Transaction, error) {
	return w.tx, nil
}

type methodsStub struct {
	MethodRepository
	tx         context.Context
	continuity bool
	writes     int
}

func (r *methodsStub) SetActive(ctx context.Context, org, code string, active bool) (bool, error) {
	if ctx != r.tx {
		return false, errors.New("wrong transaction")
	}
	r.writes++
	return true, nil
}
func (r *methodsStub) Continuity(ctx context.Context, org string) (bool, error) {
	if ctx != r.tx {
		return false, errors.New("wrong transaction")
	}
	return r.continuity, nil
}
func TestMethodsRejectRemovingLastUsableMethod(t *testing.T) {
	for _, continuity := range []bool{false, true} {
		tx := &methodTx{ctx: context.WithValue(context.Background(), struct{}{}, "transaction")}
		repo := &methodsStub{tx: tx.ctx, continuity: continuity}
		service := NewMethods(repo, methodWork{tx})
		err := service.SetActive(context.Background(), "org", "cash", false)
		if continuity {
			if err != nil || !tx.committed {
				t.Fatalf("valid change: %v", err)
			}
		} else {
			var problem *apperror.Error
			if !errors.As(err, &problem) || problem.Code != "payment_method_required" || tx.committed {
				t.Fatalf("last method: %v", err)
			}
		}
		if !tx.rolledBack || repo.writes != 1 {
			t.Fatal("unit of work was not finalized")
		}
	}
}
func TestMethodsValidateBeforeStartingTransaction(t *testing.T) {
	service := NewMethods(nil, nil)
	if _, err := service.Save(context.Background(), "org", "cash", domain.PaymentMethodInput{Code: "other", Name: "Cash", SalesEnabled: true}, true); err == nil {
		t.Fatal("allowed code change")
	}
	if _, err := service.Save(context.Background(), "org", "", domain.PaymentMethodInput{Code: "cash", Name: "Cash"}, false); err == nil {
		t.Fatal("allowed method without usage")
	}
}
