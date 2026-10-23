package http

import (
	"encoding/json"
	"net/http"

	"github.com/foods-platform/foods-backend/internal/modules/concierge/application"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
)

// UsageHandler serves the Concierge integration (conversation opening and
// token reports) and the platform-administrator usage screens. Route
// composition applies the service key or the platform-admin check.
type UsageHandler struct{ service *application.Usage }

func NewUsage(service *application.Usage) *UsageHandler { return &UsageHandler{service} }

func (h *UsageHandler) OpenConciergeConversation(w http.ResponseWriter, r *http.Request) {
	var in domain.ConversationInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_conversation", "Revisa la conversación enviada.")
		return
	}
	out, created, err := h.service.OpenConversation(r.Context(), r.PathValue("token"), in)
	status := 200
	if created {
		status = 201
	}
	respond(w, status, out, err)
}

func (h *UsageHandler) ReportConciergeUsage(w http.ResponseWriter, r *http.Request) {
	var in domain.UsageReport
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_usage_report", "Revisa el consumo enviado.")
		return
	}
	err := h.service.ReportUsage(r.Context(), r.PathValue("token"), r.PathValue("conversationId"), in)
	respond(w, 204, nil, err)
}

func (h *UsageHandler) GetConciergeUsageSettings(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Settings(r.Context())
	respond(w, 200, out, err)
}

func (h *UsageHandler) UpdateConciergeUsageSettings(w http.ResponseWriter, r *http.Request) {
	var in domain.UsageSettingsInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_concierge_usage_settings", "Revisa la configuración enviada.")
		return
	}
	out, err := h.service.SaveSettings(r.Context(), conciergeScope(r), in)
	respond(w, 200, out, err)
}

func (h *UsageHandler) ListConciergeUsage(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.List(r.Context(), r.URL.Query()["organizationId"])
	respond(w, 200, out, err)
}

func (h *UsageHandler) GetOrganizationConciergeUsage(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Organization(r.Context(), r.PathValue("id"))
	respond(w, 200, out, err)
}

func (h *UsageHandler) UpdateOrganizationConciergeLimit(w http.ResponseWriter, r *http.Request) {
	var in domain.LimitOverrideInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_concierge_limit", "Revisa el tope enviado.")
		return
	}
	out, err := h.service.SaveOverride(r.Context(), conciergeScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
