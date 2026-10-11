package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/reporting/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"testing"
)

type reportTx struct {
	ctx                   context.Context
	committed, rolledBack bool
}

func (t *reportTx) Context() context.Context       { return t.ctx }
func (t *reportTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *reportTx) Rollback(context.Context) error { t.rolledBack = true; return nil }

type reportWork struct {
	tx      *reportTx
	options unitofwork.Options
}

func (w *reportWork) Begin(ctx context.Context, options unitofwork.Options) (unitofwork.Transaction, error) {
	w.options = options
	return w.tx, nil
}

type reportAuthorization struct {
	visible bool
	err     error
}

func (a reportAuthorization) CashExpectedVisible(context.Context, requestscope.Scope) (bool, error) {
	return a.visible, a.err
}

type reportRepo struct {
	ctx     context.Context
	visible bool
	err     error
}

func (r *reportRepo) Dashboard(ctx context.Context, s requestscope.Scope, period domain.DashboardPeriod, visible bool, labels map[string]string) (domain.Dashboard, error) {
	r.ctx = ctx
	r.visible = visible
	return domain.Dashboard{}, r.err
}
func TestDashboardKeepsSnapshotAndFailsClosedForCash(t *testing.T) {
	for _, failed := range []bool{false, true} {
		tx := &reportTx{ctx: context.WithValue(context.Background(), struct{}{}, "snapshot")}
		work := &reportWork{tx: tx}
		repo := &reportRepo{}
		authorization := reportAuthorization{visible: true}
		if failed {
			authorization.err = errors.New("permission read failed")
		}
		_, err := New(repo, work, authorization).Dashboard(context.Background(), requestscope.Scope{}, "7d")
		if err != nil {
			t.Fatal(err)
		}
		if repo.ctx != tx.ctx || !work.options.ReadOnly || !work.options.RepeatableRead {
			t.Fatal("report escaped its read-only snapshot")
		}
		if repo.visible == failed {
			t.Fatal("unexpected cash visibility")
		}
		if !tx.committed || !tx.rolledBack {
			t.Fatal("snapshot was not finalized")
		}
	}
}
func TestDashboardRollsBackFailedProjection(t *testing.T) {
	tx := &reportTx{ctx: context.Background()}
	work := &reportWork{tx: tx}
	repo := &reportRepo{err: &ReportError{Stage: Hourly, Cause: errors.New("query failed")}}
	_, err := New(repo, work, reportAuthorization{}).Dashboard(context.Background(), requestscope.Scope{}, "today")
	var problem *apperror.Error
	if !errors.As(err, &problem) || problem.Code != "dashboard_unavailable" || problem.Message != "No pudimos cargar las ventas por hora." {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatal("failed report transaction leaked")
	}
}
func TestDashboardRejectsPeriodBeforeRepositories(t *testing.T) {
	_, err := New(nil, nil, nil).Dashboard(context.Background(), requestscope.Scope{}, "invalid")
	var problem *apperror.Error
	if !errors.As(err, &problem) || problem.Code != "invalid_period" {
		t.Fatalf("unexpected error: %v", err)
	}
}
