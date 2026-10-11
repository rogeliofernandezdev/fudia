package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type Expenses struct{ service *application.Expenses }

func NewExpenses(service *application.Expenses) *Expenses { return &Expenses{service} }
func (h *Expenses) ListExpenses(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	out, err := h.service.List(r.Context(), billingScope(r), application.ExpenseFilter{Query: q.Get("q"), CategoryID: q.Get("categoryId"), Status: q.Get("status"), From: q.Get("from"), To: q.Get("to"), Page: page, Size: size})
	billingResponse(w, 200, out, err)
}
func (h *Expenses) ListExpenseCategories(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	out, err := h.service.Categories(r.Context(), billingScope(r).OrganizationID, application.MethodFilter{Query: q.Get("q"), Status: q.Get("status"), Page: page, Size: size})
	billingResponse(w, 200, out, err)
}
func (h *Expenses) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var in domain.ExpenseInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_expense", "Revisa los datos del gasto.")
		return
	}
	out, err := h.service.Create(r.Context(), billingScope(r), in)
	billingResponse(w, 201, out, err)
}
func (h *Expenses) VoidExpense(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reason string `json:"reason"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_expense_void", "Indica el motivo de anulación.")
		return
	}
	out, err := h.service.Void(r.Context(), billingScope(r), r.PathValue("id"), in.Reason)
	billingResponse(w, 200, out, err)
}
func (h *Expenses) CreateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	h.saveCategory(w, r, false)
}
func (h *Expenses) UpdateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	h.saveCategory(w, r, true)
}
func (h *Expenses) saveCategory(w http.ResponseWriter, r *http.Request, updating bool) {
	var in domain.ExpenseCategoryInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_expense_category", "Revisa el nombre de la categoría.")
		return
	}
	out, err := h.service.SaveCategory(r.Context(), billingScope(r), r.PathValue("id"), in, updating)
	status := 201
	if updating {
		status = 200
	}
	billingResponse(w, status, out, err)
}
func (h *Expenses) UpdateExpenseCategoryStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_expense_category_status", "El estado no es válido.")
		return
	}
	err := h.service.SetCategoryActive(r.Context(), billingScope(r), r.PathValue("id"), in.Active)
	billingResponse(w, 204, nil, err)
}
