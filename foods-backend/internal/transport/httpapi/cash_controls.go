package httpapi

import (
	"net/http"
)

func (a *API) assignCashShiftUser(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.AssignCashShiftUser(w, r)
}
func (a *API) createCashOperation(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.CreateCashOperation(w, r)
}
func (a *API) listCashUserOptions(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.ListCashUserOptions(w, r)
}
func (a *API) listCashShiftUsers(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.ListCashShiftUsers(w, r)
}
func (a *API) unassignCashShiftUser(w http.ResponseWriter, r *http.Request) {
	a.handlers.Cash.UnassignCashShiftUser(w, r)
}
