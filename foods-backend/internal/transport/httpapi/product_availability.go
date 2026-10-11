package httpapi

import (
	"net/http"
)

func (a *API) listProductAvailability(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListProductAvailability(w, r)
}
func (a *API) updateProductAvailability(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.UpdateProductAvailability(w, r)
}
func (a *API) listProductAvailabilityHistory(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListProductAvailabilityHistory(w, r)
}
