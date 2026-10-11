package httpapi

import (
	"net/http"
)

type ExpensesHandler interface {
	ListExpenses(http.ResponseWriter, *http.Request)
	CreateExpense(http.ResponseWriter, *http.Request)
	VoidExpense(http.ResponseWriter, *http.Request)
	ListExpenseCategories(http.ResponseWriter, *http.Request)
	CreateExpenseCategory(http.ResponseWriter, *http.Request)
	UpdateExpenseCategory(http.ResponseWriter, *http.Request)
	UpdateExpenseCategoryStatus(http.ResponseWriter, *http.Request)
}

func (a *API) listExpenses(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.ListExpenses(w, r)
}
func (a *API) createExpense(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.CreateExpense(w, r)
}
func (a *API) voidExpense(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.VoidExpense(w, r)
}
func (a *API) listExpenseCategories(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.ListExpenseCategories(w, r)
}
func (a *API) createExpenseCategory(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.CreateExpenseCategory(w, r)
}
func (a *API) updateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.UpdateExpenseCategory(w, r)
}
func (a *API) updateExpenseCategoryStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Expenses.UpdateExpenseCategoryStatus(w, r)
}
