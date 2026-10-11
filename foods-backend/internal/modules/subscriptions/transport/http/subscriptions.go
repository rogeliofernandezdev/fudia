package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Subscriptions struct{ service *application.Subscriptions }

func New(service *application.Subscriptions) *Subscriptions { return &Subscriptions{service} }
func subscriptionScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}
func subscriptionResponse(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, status, out)
}
func (h *Subscriptions) CreateSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	var in domain.SubscriptionPlanInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_plan", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SavePlan(r.Context(), subscriptionScope(r), r.PathValue("id"), in, true)
	subscriptionResponse(w, 201, out, err)
}
func (h *Subscriptions) UpdateSubscriptionPlan(w http.ResponseWriter, r *http.Request) {
	var in domain.SubscriptionPlanInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_plan", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SavePlan(r.Context(), subscriptionScope(r), r.PathValue("id"), in, false)
	subscriptionResponse(w, 200, out, err)
}
func (h *Subscriptions) UpdateOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	var in domain.SubscriptionUpdateInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_subscription", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Update(r.Context(), subscriptionScope(r), r.PathValue("id"), in)
	subscriptionResponse(w, 200, out, err)
}
func (h *Subscriptions) RecordSubscriptionPayment(w http.ResponseWriter, r *http.Request) {
	var in domain.SubscriptionPaymentInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_payment", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.RecordPayment(r.Context(), subscriptionScope(r), r.PathValue("id"), in)
	subscriptionResponse(w, 201, out, err)
}
func (h *Subscriptions) ListSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Plans(r.Context())
	subscriptionResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Subscriptions) ListAvailableSubscriptionPlans(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.AvailablePlans(r.Context())
	subscriptionResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Subscriptions) GetOrganizationSubscription(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Get(r.Context(), subscriptionScope(r), r.PathValue("id"))
	subscriptionResponse(w, 200, out, err)
}
