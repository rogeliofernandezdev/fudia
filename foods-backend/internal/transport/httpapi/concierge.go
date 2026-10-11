package httpapi

import (
	"net/http"
)

func (a *API) conciergeServiceAuth(next http.Handler) http.Handler {
	return a.handlers.Concierge.ServiceAuth(next)
}
func (a *API) createConciergeOrder(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.CreateConciergeOrder(w, r)
}
func (a *API) requestConciergeBill(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.RequestConciergeBill(w, r)
}
func (a *API) requestConciergeHandoff(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.RequestConciergeHandoff(w, r)
}
func (a *API) getConciergeHandoffStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.GetConciergeHandoffStatus(w, r)
}
func (a *API) listOperationalConciergeHandoffs(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.ListOperationalConciergeHandoffs(w, r)
}
func (a *API) resolveOperationalConciergeHandoff(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.ResolveOperationalConciergeHandoff(w, r)
}
func (a *API) getConciergeSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.GetConciergeSettings(w, r)
}
func (a *API) listConciergeMenu(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.ListConciergeMenu(w, r)
}
func (a *API) getConciergeCombo(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.GetConciergeCombo(w, r)
}
func (a *API) getConciergeProductModifiers(w http.ResponseWriter, r *http.Request) {
	a.handlers.Concierge.GetConciergeProductModifiers(w, r)
}
