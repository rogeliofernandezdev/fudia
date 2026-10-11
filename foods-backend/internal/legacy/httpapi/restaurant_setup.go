package httpapi

import (
	orgdomain "github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"net/http"
)

type restaurantSetupCounts = orgdomain.RestaurantSetupCounts
type restaurantSetupView = orgdomain.RestaurantSetupView

var validServiceMode = orgdomain.ValidServiceMode
var restaurantSetupCoreReady = orgdomain.RestaurantSetupCoreReady

func (a *API) updateRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.UpdateRestaurantSetup(w, r)
}
func (a *API) getRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.GetRestaurantSetup(w, r)
}
func (a *API) completeRestaurantSetup(w http.ResponseWriter, r *http.Request) {
	a.handlers.Organizations.CompleteRestaurantSetup(w, r)
}
