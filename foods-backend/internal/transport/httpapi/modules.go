package httpapi

import (
	"context"

	"net/http"
)

type ModulesHandler interface {
	ListModules(http.ResponseWriter, *http.Request)
	ToggleModule(http.ResponseWriter, *http.Request)
	ActiveModules(context.Context, string) (map[string]bool, error)
}

func (a *API) listModules(w http.ResponseWriter, r *http.Request) {
	a.handlers.Modules.ListModules(w, r)
}
func (a *API) toggleModule(w http.ResponseWriter, r *http.Request) {
	a.handlers.Modules.ToggleModule(w, r)
}
