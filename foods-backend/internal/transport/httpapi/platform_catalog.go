package httpapi

import (
	"net/http"
)

func (a *API) getPlatformOnboardingCatalogs(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.GetPlatformOnboardingCatalogs(w, r)
}
