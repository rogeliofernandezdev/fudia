package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

var ErrNotFound = errors.New("billing record not found")

type MethodFilter struct {
	Query, Status string
	Page, Size    int
}
type MethodPage struct {
	Items    []domain.PaymentMethodView `json:"items"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}
type MethodRepository interface {
	Operational(context.Context, string, string) ([]domain.PaymentMethodView, error)
	List(context.Context, string, MethodFilter) (MethodPage, error)
	Save(context.Context, string, string, domain.PaymentMethodInput, bool) (domain.PaymentMethodView, error)
	SetActive(context.Context, string, string, bool) (bool, error)
	Continuity(context.Context, string) (bool, error)
	// ActiveCount locks the organization's catalog and counts its active methods.
	ActiveCount(context.Context, string) (int, error)
	Find(context.Context, string, string, string) (domain.PaymentMethodView, error)
}
type Methods struct {
	repo MethodRepository
	work unitofwork.Manager
}

func NewMethods(repo MethodRepository, work unitofwork.Manager) *Methods { return &Methods{repo, work} }
func methodFailure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}
func (m *Methods) Operational(ctx context.Context, org, usage string) ([]domain.PaymentMethodView, error) {
	usage = strings.TrimSpace(usage)
	if usage != "sales" && usage != "expenses" {
		return nil, methodFailure(apperror.Invalid, "invalid_payment_method_usage", "Indica si necesitas medios para ventas o gastos.")
	}
	items, err := m.repo.Operational(ctx, org, usage)
	if err != nil {
		return nil, methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
	}
	return items, nil
}
func (m *Methods) Options(ctx context.Context, org, usage string) ([]map[string]string, error) {
	if usage != "sales" && usage != "expenses" {
		return []map[string]string{}, nil
	}
	items, err := m.Operational(ctx, org, usage)
	if err != nil {
		return nil, err
	}
	out := []map[string]string{}
	for _, v := range items {
		out = append(out, map[string]string{"value": v.Code, "label": v.Name})
	}
	return out, nil
}
func (m *Methods) Find(ctx context.Context, org, code, usage string) (domain.PaymentMethodView, error) {
	return m.repo.Find(ctx, org, strings.TrimSpace(code), strings.TrimSpace(usage))
}
func (m *Methods) List(ctx context.Context, org string, f MethodFilter) (MethodPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	if f.Status != "" && f.Status != "active" && f.Status != "inactive" {
		return MethodPage{}, methodFailure(apperror.Invalid, "invalid_payment_method_filter", "El estado del medio de pago no es válido.")
	}
	out, err := m.repo.List(ctx, org, f)
	if err != nil {
		return out, methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
	}
	return out, nil
}
func (m *Methods) continuity(ctx context.Context, org string) error {
	ok, err := m.repo.Continuity(ctx, org)
	if err != nil {
		return methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos validar la configuración de medios de pago.")
	}
	if !ok {
		return methodFailure(apperror.Conflict, "payment_method_required", "Debe quedar al menos un medio activo para ventas y uno para gastos.")
	}
	return nil
}
func (m *Methods) activeLimit(ctx context.Context, org string) error {
	count, err := m.repo.ActiveCount(ctx, org)
	if err != nil {
		return methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos validar la cantidad de medios de pago activos.")
	}
	if count > domain.MaxActivePaymentMethods {
		return methodFailure(apperror.Conflict, "payment_method_limit", "Puedes tener hasta 4 medios de pago activos. Desactiva uno antes de agregar o activar otro.")
	}
	return nil
}
func (m *Methods) Save(ctx context.Context, org, code string, in domain.PaymentMethodInput, updating bool) (domain.PaymentMethodView, error) {
	var out domain.PaymentMethodView
	domain.NormalizePaymentMethodInput(&in)
	if updating {
		code = strings.TrimSpace(code)
		if !domain.PaymentMethodCodePattern.MatchString(code) || (in.Code != "" && in.Code != code) {
			return out, methodFailure(apperror.Invalid, "immutable_payment_method_code", "El código interno no puede modificarse.")
		}
	} else {
		code = in.Code
	}
	if !domain.ValidPaymentMethodInput(in, !updating) {
		return out, methodFailure(apperror.Invalid, "invalid_payment_method", "Completa un código, nombre, uso y orden válidos.")
	}
	tx, err := m.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos guardar el medio de pago.")
	}
	defer tx.Rollback(ctx)
	out, err = m.repo.Save(tx.Context(), org, code, in, updating)
	if errors.Is(err, ErrNotFound) {
		return out, methodFailure(apperror.NotFound, "payment_method_not_found", "El medio de pago no existe.")
	}
	if err != nil {
		return out, methodFailure(apperror.Conflict, "payment_method_conflict", "Ya existe un medio de pago con ese código.")
	}
	if err = m.continuity(tx.Context(), org); err != nil {
		return out, err
	}
	if !updating {
		if err = m.activeLimit(tx.Context(), org); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos guardar el medio de pago.")
	}
	return out, nil
}
func (m *Methods) SetActive(ctx context.Context, org, code string, active bool) error {
	code = strings.TrimSpace(code)
	if !domain.PaymentMethodCodePattern.MatchString(code) {
		return methodFailure(apperror.NotFound, "payment_method_not_found", "El medio de pago no existe.")
	}
	tx, err := m.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos actualizar el medio de pago.")
	}
	defer tx.Rollback(ctx)
	changed, err := m.repo.SetActive(tx.Context(), org, code, active)
	if err != nil {
		return methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos actualizar el medio de pago.")
	}
	if !changed {
		return methodFailure(apperror.NotFound, "payment_method_not_found", "El medio de pago no existe.")
	}
	if err = m.continuity(tx.Context(), org); err != nil {
		return err
	}
	if active {
		if err = m.activeLimit(tx.Context(), org); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos actualizar el medio de pago.")
	}
	return nil
}
