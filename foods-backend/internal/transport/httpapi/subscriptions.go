package httpapi

import (
	"net/http"
)

func (a *API) listSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.ListSubscriptionPlans(w, r)
}
func (a *API) listAvailableSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.ListAvailableSubscriptionPlans(w, r)
}
func (a *API) createSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.CreateSubscriptionPlan(w, r)
}
func (a *API) updateSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.UpdateSubscriptionPlan(w, r)
}
func (a *API) getOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.GetOrganizationSubscription(w, r)
}
func (a *API) updateOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.UpdateOrganizationSubscription(w, r)
}
func (a *API) recordSubscriptionPayment(w http.ResponseWriter, r *http.Request) {
	a.handlers.Subscriptions.RecordSubscriptionPayment(w, r)
}
