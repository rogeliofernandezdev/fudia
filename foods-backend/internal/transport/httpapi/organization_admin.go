package httpapi

import (
	"net/http"
)

func (a *API) updateOrganization(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateOrganization(w, r)
}
func (a *API) createLocation(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.CreateLocation(w, r)
}
func (a *API) updateLocation(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateLocation(w, r)
}
func (a *API) createExchangeRate(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.CreateExchangeRate(w, r)
}
func (a *API) createFiscalProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.CreateFiscalProfile(w, r)
}
func (a *API) updateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateFiscalProfile(w, r)
}
func (a *API) getOrganization(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.GetOrganization(w, r)
}
func (a *API) deactivateLocation(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.DeactivateLocation(w, r)
}
func (a *API) deactivateFiscalProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.DeactivateFiscalProfile(w, r)
}
func (a *API) deleteExchangeRate(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.DeleteExchangeRate(w, r)
}
func (a *API) listLocations(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.ListLocations(w, r)
}
func (a *API) listFiscalProfiles(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.ListFiscalProfiles(w, r)
}
func (a *API) listExchangeRates(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.ListExchangeRates(w, r)
}
