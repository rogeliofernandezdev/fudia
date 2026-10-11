package httpapi

import (
	"net/http"
)

func (a *API) listPlatformWhatsAppChannels(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.ListPlatformWhatsAppChannels(w, r)
}
func (a *API) createPlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.CreatePlatformWhatsAppChannel(w, r)
}
func (a *API) updatePlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.UpdatePlatformWhatsAppChannel(w, r)
}
