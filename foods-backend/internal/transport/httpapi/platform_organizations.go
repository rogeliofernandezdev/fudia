package httpapi

import (
	"net/http"
)

func (a *API) listPlatformOrganizations(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.ListPlatformOrganizations(w, r)
}
