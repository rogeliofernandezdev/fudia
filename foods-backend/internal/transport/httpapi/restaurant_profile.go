package httpapi

import "net/http"

func (a *API) getRestaurantProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.RestaurantProfile.RestaurantProfile(w, r)
}
func (a *API) saveRestaurantProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.RestaurantProfile.SaveRestaurantProfile(w, r)
}
func (a *API) authorizeRestaurantImage(w http.ResponseWriter, r *http.Request) {
	a.handlers.RestaurantProfile.AuthorizeRestaurantImage(w, r)
}
func (a *API) completeRestaurantImage(w http.ResponseWriter, r *http.Request) {
	a.handlers.RestaurantProfile.CompleteRestaurantImage(w, r)
}
func (a *API) getRestaurantImage(w http.ResponseWriter, r *http.Request) {
	a.handlers.RestaurantProfile.RestaurantImage(w, r)
}
func (a *API) getRestaurantBranding(w http.ResponseWriter, r *http.Request) {
	a.handlers.RestaurantProfile.RestaurantBranding(w, r)
}
