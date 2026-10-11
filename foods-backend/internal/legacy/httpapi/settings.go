package httpapi

import (
	orgdomain "github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"net/http"
)

type orgSettings = orgdomain.OrgSettings
type orgSettingsInput = orgdomain.OrgSettingsInput

var countryByCode = orgdomain.CountryByCode
var currencyByCode = orgdomain.CurrencyByCode

func (a *API) updateOrgSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateOrgSettings(w, r)
}
func (a *API) getOrgSettings(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.GetOrgSettings(w, r)
}
