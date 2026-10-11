package httpapi

import (
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"net/http"
)

type expenseCategoryView = billingdomain.ExpenseCategoryView
type expenseView = billingdomain.ExpenseView
type expenseInput = billingdomain.ExpenseInput

var expenseAmountPattern = billingdomain.ExpenseAmountPattern
var validExpenseAmount = billingdomain.ValidExpenseAmount

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
func (a *API) saveExpenseCategory(w http.ResponseWriter, r *http.Request, updating bool) {
	if updating {
		a.updateExpenseCategory(w, r)
	} else {
		a.createExpenseCategory(w, r)
	}
}
