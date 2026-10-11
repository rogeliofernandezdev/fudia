package application

import (
	"context"
	"errors"
	ordercontracts "github.com/foods-platform/foods-backend/internal/modules/orders/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/reporting/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type Stage uint8

const (
	Summary Stage = iota
	Hourly
	Products
	Operations
	Insights
)

type ReportError struct {
	Stage Stage
	Cause error
}

func (e *ReportError) Error() string { return e.Cause.Error() }
func (e *ReportError) Unwrap() error { return e.Cause }

type Repository interface {
	Dashboard(context.Context, requestscope.Scope, domain.DashboardPeriod, bool, map[string]string) (domain.Dashboard, error)
}
type Authorizer interface {
	CashExpectedVisible(context.Context, requestscope.Scope) (bool, error)
}
type Service struct {
	repo       Repository
	work       unitofwork.Manager
	authorizer Authorizer
}

func New(repo Repository, work unitofwork.Manager, authorizer Authorizer) *Service {
	return &Service{repo, work, authorizer}
}
func (s *Service) Dashboard(ctx context.Context, scope requestscope.Scope, key string) (domain.Dashboard, error) {
	var out domain.Dashboard
	period, ok := domain.DashboardPeriodByKey(key)
	if !ok {
		return out, apperror.New(apperror.Invalid, "invalid_period", "El periodo del resumen no es válido.")
	}
	visible, permissionErr := s.authorizer.CashExpectedVisible(ctx, scope)
	if permissionErr != nil {
		visible = false
	}
	tx, err := s.work.Begin(ctx, unitofwork.Options{ReadOnly: true, RepeatableRead: true})
	if err != nil {
		return out, apperror.New(apperror.Unavailable, "dashboard_unavailable", "No pudimos calcular el resumen operativo.")
	}
	defer tx.Rollback(ctx)
	labels := map[string]string{}
	for _, v := range ordercontracts.Channels {
		labels[v["value"]] = v["label"]
	}
	out, err = s.repo.Dashboard(tx.Context(), scope, period, visible, labels)
	if err != nil {
		message := "No pudimos calcular el resumen operativo."
		var failure *ReportError
		if errors.As(err, &failure) {
			switch failure.Stage {
			case Hourly:
				message = "No pudimos cargar las ventas por hora."
			case Products:
				message = "No pudimos cargar los productos vendidos."
			case Operations:
				message = "No pudimos calcular el estado del restaurante."
			case Insights:
				message = "No pudimos calcular las tendencias de ventas."
			}
		}
		return out, apperror.New(apperror.Unavailable, "dashboard_unavailable", message)
	}
	if err = tx.Commit(ctx); err != nil {
		return out, apperror.New(apperror.Unavailable, "dashboard_unavailable", "No pudimos completar el resumen operativo.")
	}
	return out, nil
}
