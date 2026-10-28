package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/billing/application"
	"github.com/foods-platform/foods-backend/internal/modules/billing/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type Tips struct{ service *application.Tips }

func NewTips(service *application.Tips) *Tips { return &Tips{service} }
func (h *Tips) GetSettings(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Policy(r.Context(), billingScope(r), r.URL.Query().Get("scope") == "organization")
	billingResponse(w, 200, out, err)
}
func (h *Tips) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in domain.TipSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_tip_settings", "Revisa la configuración de propinas.")
		return
	}
	out, err := h.service.SavePolicy(r.Context(), billingScope(r), r.URL.Query().Get("scope") == "organization", in)
	billingResponse(w, 200, out, err)
}
func (h *Tips) Workspace(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Workspace(r.Context(), billingScope(r), r.PathValue("id"))
	billingResponse(w, 200, out, err)
}
func (h *Tips) Payout(w http.ResponseWriter, r *http.Request) {
	var in domain.TipPayoutInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_tip_payout", "Revisa los datos de la entrega de propinas.")
		return
	}
	out, err := h.service.Payout(r.Context(), billingScope(r), r.PathValue("id"), in)
	billingResponse(w, 201, out, err)
}
