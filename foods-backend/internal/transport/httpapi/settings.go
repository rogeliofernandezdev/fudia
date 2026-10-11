package httpapi

import (
	"net/http"
)

func (a *API) updateOrgSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateOrgSettings(w, r)
}
func (a *API) getOrgSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.GetOrgSettings(w, r)
}
