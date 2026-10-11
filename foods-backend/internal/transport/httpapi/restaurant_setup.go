package httpapi

import (
	"net/http"
)

func (a *API) updateRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateRestaurantSetup(w, r)
}
func (a *API) getRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.GetRestaurantSetup(w, r)
}
func (a *API) completeRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.CompleteRestaurantSetup(w, r)
}
