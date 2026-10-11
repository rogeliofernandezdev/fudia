package httpapi

import (
	"net/http"
)

func (a *API) listRecipes(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.ListRecipes(w, r)
}
func (a *API) getRecipe(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.GetRecipe(w, r)
}
func (a *API) saveRecipe(w http.ResponseWriter, r *http.Request) {
	a.handlers.MenuCatalog.SaveRecipe(w, r)
}
