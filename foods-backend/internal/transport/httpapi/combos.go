package httpapi

import (
	"net/http"
)

func (a *API) listCombos(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListCombos(w, r)
}
func (a *API) listOrderCombos(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListOrderCombos(w, r)
}
func (a *API) getCombo(w http.ResponseWriter, r *http.Request) { a.handlers.MenuCatalog.GetCombo(w, r) }
func (a *API) getOrderCombo(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.GetOrderCombo(w, r)
}
func (a *API) saveCombo(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.SaveCombo(w, r)
}
func (a *API) updateComboStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateComboStatus(w, r)
}
