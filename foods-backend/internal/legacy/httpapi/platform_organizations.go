package httpapi

import (
	platformdomain "github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"net/http"
)

type platformOrganizationPlan = platformdomain.PlatformOrganizationPlan
type platformOrganizationSubscription = platformdomain.PlatformOrganizationSubscription
type platformOrganizationPayment = platformdomain.PlatformOrganizationPayment
type platformOrganizationView = platformdomain.PlatformOrganizationView
type optionView = platformdomain.OptionView

var validPaymentStanding = platformdomain.ValidPaymentStanding

const subscriptionDueSoonDays = platformdomain.SubscriptionDueSoonDays

var paymentStandingOptions = platformdomain.PaymentStandingOptions

func (a *API) listPlatformOrganizations(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.ListPlatformOrganizations(w, r)
}
