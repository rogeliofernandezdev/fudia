package httpapi

import (
	"net/http"
)

func (a *API) getProductModifiers(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.GetProductModifiers(w, r)
}
func (a *API) saveProductModifiers(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.SaveProductModifiers(w, r)
}
