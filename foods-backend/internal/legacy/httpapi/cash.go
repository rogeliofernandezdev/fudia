package httpapi

import (
	billingdomain "github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	"net/http"
)

type cashMovementView = billingdomain.CashMovementView
type cashShiftView = billingdomain.CashShiftView
type cashRegisterView = billingdomain.CashRegisterView

func (a *API) createCashRegister(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.CreateCashRegister(w, r)
}
func (a *API) updateCashRegister(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.UpdateCashRegister(w, r)
}
func (a *API) updateCashRegisterStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.UpdateCashRegisterStatus(w, r)
}
func (a *API) openCashShift(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.OpenCashShift(w, r)
}
func (a *API) createCashMovement(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.CreateCashMovement(w, r)
}
func (a *API) closeCashShift(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.CloseCashShift(w, r)
}
func (a *API) listCashRegisters(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.ListCashRegisters(w, r)
}
func (a *API) getCurrentCashShift(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.GetCurrentCashShift(w, r)
}
func (a *API) listCashShifts(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.ListCashShifts(w, r)
}
func (a *API) getCashShift(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.GetCashShift(w, r)
}
