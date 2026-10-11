package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"strings"
	"time"
)

type ExpenseFilter struct {
	Query, CategoryID, Status, From, To string
	Page, Size                          int
}
type ExpensePage struct {
	Items                []domain.ExpenseView `json:"items"`
	Total                int                  `json:"total"`
	Page                 int                  `json:"page"`
	PageSize             int                  `json:"pageSize"`
	PaymentMethodOptions []map[string]string  `json:"paymentMethodOptions"`
}
type ExpenseCategoryPage struct {
	Items    []domain.ExpenseCategoryView `json:"items"`
	Total    int                          `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"pageSize"`
}
type ExpenseRepository interface {
	List(context.Context, requestscope.Scope, ExpenseFilter) (ExpensePage, error)
	Get(context.Context, requestscope.Scope, string) (domain.ExpenseView, error)
	CategoryActive(context.Context, string, string) (bool, error)
	Create(context.Context, requestscope.Scope, domain.ExpenseInput) (string, error)
	Status(context.Context, requestscope.Scope, string) (string, error)
	Void(context.Context, requestscope.Scope, string, string) (int64, error)
	Categories(context.Context, string, MethodFilter) (ExpenseCategoryPage, error)
	SaveCategory(context.Context, string, string, string, bool) (domain.ExpenseCategoryView, error)
	SetCategoryActive(context.Context, string, string, bool) (int64, error)
}
type ExpenseMethods interface {
	Find(context.Context, string, string, string) (domain.PaymentMethodView, error)
	Options(context.Context, string, string) ([]map[string]string, error)
}
type Expenses struct {
	repo    ExpenseRepository
	methods ExpenseMethods
	audit   audit.Repository
}

func NewExpenses(repo ExpenseRepository, methods ExpenseMethods, audit audit.Repository) *Expenses {
	return &Expenses{repo, methods, audit}
}
func (a *Expenses) List(ctx context.Context, s requestscope.Scope, f ExpenseFilter) (ExpensePage, error) {
	var out ExpensePage
	f.Query = strings.TrimSpace(f.Query)
	f.CategoryID = strings.TrimSpace(f.CategoryID)
	f.Status = strings.TrimSpace(f.Status)
	f.From = strings.TrimSpace(f.From)
	f.To = strings.TrimSpace(f.To)
	if f.Status != "" && f.Status != "active" && f.Status != "void" {
		return out, methodFailure(apperror.Invalid, "invalid_expense_filter", "El estado de gasto no es válido.")
	}
	for _, date := range []string{f.From, f.To} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return out, methodFailure(apperror.Invalid, "invalid_expense_filter", "Revisa el rango de fechas.")
			}
		}
	}
	out, err := a.repo.List(ctx, s, f)
	if err != nil {
		return out, methodFailure(apperror.Unavailable, "expenses_unavailable", "No pudimos cargar los gastos.")
	}
	out.PaymentMethodOptions, err = a.methods.Options(ctx, s.OrganizationID, "expenses")
	if err != nil {
		return out, methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos cargar los medios de pago.")
	}
	return out, nil
}
func (a *Expenses) Categories(ctx context.Context, org string, f MethodFilter) (ExpenseCategoryPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.Status = strings.TrimSpace(f.Status)
	if f.Status != "" && f.Status != "active" && f.Status != "inactive" {
		return ExpenseCategoryPage{}, methodFailure(apperror.Invalid, "invalid_expense_category_filter", "El estado de categoría no es válido.")
	}
	out, err := a.repo.Categories(ctx, org, f)
	if err != nil {
		return out, methodFailure(apperror.Unavailable, "expense_categories_unavailable", "No pudimos cargar las categorías.")
	}
	return out, nil
}
func (a *Expenses) Create(ctx context.Context, s requestscope.Scope, in domain.ExpenseInput) (domain.ExpenseView, error) {
	var result domain.ExpenseView
	in.CategoryID = strings.TrimSpace(in.CategoryID)
	in.Description = strings.TrimSpace(in.Description)
	in.Amount = strings.TrimSpace(in.Amount)
	in.PaymentMethod = strings.TrimSpace(in.PaymentMethod)
	in.BusinessDate = strings.TrimSpace(in.BusinessDate)
	in.Reference = strings.TrimSpace(in.Reference)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.CategoryID == "" || in.Description == "" || len(in.Description) > 180 || !domain.ValidExpenseAmount(in.Amount) || in.PaymentMethod == "" || len(in.Reference) > 120 || len(in.Notes) > 500 {
		return result, methodFailure(apperror.Invalid, "invalid_expense", "Completa categoría, descripción, importe y medio de pago válidos.")
	}
	if _, err := time.Parse("2006-01-02", in.BusinessDate); err != nil {
		return result, methodFailure(apperror.Invalid, "invalid_expense", "La fecha del gasto no es válida.")
	}
	if _, err := a.methods.Find(ctx, s.OrganizationID, in.PaymentMethod, "expenses"); errors.Is(err, ErrNotFound) {
		return result, methodFailure(apperror.Invalid, "invalid_payment_method", "Selecciona un medio de pago activo para gastos.")
	} else if err != nil {
		return result, methodFailure(apperror.Unavailable, "payment_methods_unavailable", "No pudimos validar el medio de pago.")
	}
	var categoryActive bool
	err := func() error {
		v, e := a.repo.CategoryActive(ctx, s.OrganizationID, in.CategoryID)
		categoryActive = v
		return e
	}()
	if errors.Is(err, ErrNotFound) {
		return result, methodFailure(apperror.Invalid, "expense_category_not_found", "La categoría seleccionada no existe.")
	}
	if err != nil {
		return result, methodFailure(apperror.Unavailable, "expenses_unavailable", "No pudimos validar la categoría.")
	}
	if !categoryActive {
		return result, methodFailure(apperror.Conflict, "expense_category_inactive", "Selecciona una categoría activa.")
	}
	var id string
	err = func() error { v, e := a.repo.Create(ctx, s, in); id = v; return e }()
	if err != nil {
		return result, methodFailure(apperror.Unavailable, "expenses_unavailable", "No pudimos registrar el gasto.")
	}
	_ = a.audit.Record(ctx, s, "created", "expense", id)
	item, err := a.repo.Get(ctx, s, id)
	if err != nil {
		return result, methodFailure(apperror.Unavailable, "expenses_unavailable", "El gasto se registró, pero no pudimos cargar su detalle.")
	}
	return item, nil

}
func (a *Expenses) Void(ctx context.Context, s requestscope.Scope, id, reason string) (domain.ExpenseView, error) {
	var result domain.ExpenseView
	reason = strings.TrimSpace(reason)
	if len(reason) < 4 || len(reason) > 240 {
		return result, methodFailure(apperror.Invalid, "invalid_expense_void", "El motivo debe tener entre 4 y 240 caracteres.")
	}

	var status string
	err := func() error { v, e := a.repo.Status(ctx, s, id); status = v; return e }()
	if errors.Is(err, ErrNotFound) {
		return result, methodFailure(apperror.NotFound, "expense_not_found", "El gasto no existe en este local.")
	}
	if err != nil {
		return result, methodFailure(apperror.Unavailable, "expenses_unavailable", "No pudimos cargar el gasto.")
	}
	if status == "void" {
		return result, methodFailure(apperror.Conflict, "expense_already_void", "El gasto ya está anulado.")
	}
	if _, err = a.repo.Void(ctx, s, id, reason); err != nil {
		return result, methodFailure(apperror.Unavailable, "expenses_unavailable", "No pudimos anular el gasto.")
	}
	_ = a.audit.Record(ctx, s, "voided", "expense", id)
	item, err := a.repo.Get(ctx, s, id)
	if err != nil {
		return result, methodFailure(apperror.Unavailable, "expenses_unavailable", "El gasto se anuló, pero no pudimos cargar su detalle.")
	}
	return item, nil

}
func (a *Expenses) SaveCategory(ctx context.Context, s requestscope.Scope, id string, in domain.ExpenseCategoryInput, updating bool) (domain.ExpenseCategoryView, error) {
	var result domain.ExpenseCategoryView
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 100 {
		return result, methodFailure(apperror.Invalid, "invalid_expense_category", "El nombre debe tener entre 1 y 100 caracteres.")
	}
	item, err := a.repo.SaveCategory(ctx, s.OrganizationID, id, in.Name, updating)
	if errors.Is(err, ErrNotFound) {
		return result, methodFailure(apperror.NotFound, "expense_category_not_found", "La categoría no existe.")
	}
	if err != nil {
		return result, methodFailure(apperror.Conflict, "expense_category_conflict", "Ya existe una categoría con ese nombre.")
	}
	_ = a.audit.Record(ctx, s, map[bool]string{true: "updated", false: "created"}[updating], "expense_category", item.ID)
	return item, nil

}
func (a *Expenses) SetCategoryActive(ctx context.Context, s requestscope.Scope, id string, active bool) error {
	tag, err := a.repo.SetCategoryActive(ctx, s.OrganizationID, id, active)
	if err != nil || tag == 0 {
		return methodFailure(apperror.NotFound, "expense_category_not_found", "La categoría no existe.")
	}
	_ = a.audit.Record(ctx, s, "status_updated", "expense_category", id)
	return nil

}
