package httpapi

import (
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"net/http"
)

type platformOrganizationUsage = platformdomain.PlatformOrganizationUsage
type platformOrganizationDetailSubscription = platformdomain.PlatformOrganizationDetailSubscription
type platformOrganizationPayments = platformdomain.PlatformOrganizationPayments
type platformOrganizationAdmin = platformdomain.PlatformOrganizationAdmin
type platformOrganizationLocation = platformdomain.PlatformOrganizationLocation
type platformOrganizationDetail = platformdomain.PlatformOrganizationDetail

func (a *API) getPlatformOrganization(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.GetPlatformOrganization(w, r)
}
