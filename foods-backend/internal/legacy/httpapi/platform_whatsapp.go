package httpapi

import (
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"net/http"
)

type platformWhatsAppChannel = platformdomain.PlatformWhatsAppChannel
type platformWhatsAppChannelInput = platformdomain.PlatformWhatsAppChannelInput

var validatePlatformWhatsAppChannel = platformdomain.ValidatePlatformWhatsAppChannel

func (a *API) listPlatformWhatsAppChannels(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.ListPlatformWhatsAppChannels(w, r)
}
func (a *API) createPlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.CreatePlatformWhatsAppChannel(w, r)
}
func (a *API) updatePlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.UpdatePlatformWhatsAppChannel(w, r)
}
