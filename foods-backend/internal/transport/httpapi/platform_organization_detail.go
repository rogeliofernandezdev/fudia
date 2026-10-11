package httpapi

import (
	"net/http"
)

func (a *API) getPlatformOrganization(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.GetPlatformOrganization(w, r)
}
