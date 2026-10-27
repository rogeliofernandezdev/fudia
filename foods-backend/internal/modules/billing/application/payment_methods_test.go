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
	active     int
	catalog    int
	countError error
	countCalls int
}

func (r *methodsStub) CatalogCount(ctx context.Context, org string) (int, error) {
	if ctx != r.tx {
		return 0, errors.New("wrong transaction")
	}
	r.countCalls++
	return r.catalog, r.countError
}

func (r *methodsStub) Save(ctx context.Context, org, code string, in domain.PaymentMethodInput, updating bool) (domain.PaymentMethodView, error) {
	if ctx != r.tx {
		return domain.PaymentMethodView{}, errors.New("wrong transaction")
	}
	r.writes++
	return domain.PaymentMethodView{Code: code, Name: in.Name, Active: true, SalesEnabled: in.SalesEnabled}, nil
}

func (r *methodsStub) ActiveCount(ctx context.Context, org string) (int, error) {
	if ctx != r.tx {
		return 0, errors.New("wrong transaction")
	}
	return r.active, nil
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

func TestMethodsAllowAtMostFourActive(t *testing.T) {
	for _, active := range []int{4, 5} {
		tx := &methodTx{ctx: context.WithValue(context.Background(), struct{}{}, "transaction")}
		repo := &methodsStub{tx: tx.ctx, continuity: true, active: active}
		err := NewMethods(repo, methodWork{tx}).SetActive(context.Background(), "org", "yape", true)
		var problem *apperror.Error
		if active == 4 && (err != nil || !tx.committed) {
			t.Fatalf("four active methods are allowed: %v", err)
		}
		if active == 5 && (!errors.As(err, &problem) || problem.Code != "payment_method_limit" || tx.committed) {
			t.Fatalf("a fifth active method must be rejected: %v", err)
		}
	}
}

func TestMethodsCountInactiveEntriesBeforeCreating(t *testing.T) {
	for _, count := range []int{3, 4, 5} {
		tx := &methodTx{ctx: context.WithValue(context.Background(), struct{}{}, "transaction")}
		repo := &methodsStub{tx: tx.ctx, continuity: true, catalog: count, active: 3}
		_, err := NewMethods(repo, methodWork{tx}).Save(context.Background(), "org", "", domain.PaymentMethodInput{Code: "voucher", Name: "Vale", SalesEnabled: true}, false)
		if count < 4 {
			if err != nil || !tx.committed || repo.writes != 1 {
				t.Fatalf("fourth entry rejected: %v", err)
			}
		} else {
			var problem *apperror.Error
			if !errors.As(err, &problem) || problem.Code != "payment_method_limit" || tx.committed || repo.writes != 0 {
				t.Fatalf("catalog of %d allowed another entry: %v", count, err)
			}
		}
	}
}

func TestMethodsAllowEditingFullCatalogAndFailClosedOnCountError(t *testing.T) {
	tx := &methodTx{ctx: context.WithValue(context.Background(), struct{}{}, "transaction")}
	repo := &methodsStub{tx: tx.ctx, continuity: true, catalog: 4}
	_, err := NewMethods(repo, methodWork{tx}).Save(context.Background(), "org", "cash", domain.PaymentMethodInput{Name: "Efectivo", SalesEnabled: true}, true)
	if err != nil || !tx.committed || repo.writes != 1 || repo.countCalls != 0 {
		t.Fatalf("editing full catalog: %v", err)
	}
	tx = &methodTx{ctx: context.WithValue(context.Background(), struct{}{}, "transaction")}
	repo = &methodsStub{tx: tx.ctx, continuity: true, countError: errors.New("database unavailable")}
	_, err = NewMethods(repo, methodWork{tx}).Save(context.Background(), "org", "", domain.PaymentMethodInput{Code: "voucher", Name: "Vale", SalesEnabled: true}, false)
	var problem *apperror.Error
	if !errors.As(err, &problem) || problem.Code != "payment_methods_unavailable" || tx.committed || repo.writes != 0 {
		t.Fatalf("count failure permitted write: %v", err)
	}
}
