package http

import (
	"context"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/observability"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Handler struct{ service *application.Platform }

func New(service *application.Platform) *Handler { return &Handler{service} }
func platformScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}
func respond(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, status, out)
}
func (h *Handler) Authorize(ctx context.Context, id string) error {
	return h.service.Authorize(ctx, id)
}
func (h *Handler) ListPlatformOrganizations(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	q := r.URL.Query()
	out, err := h.service.Organizations(r.Context(), platformScope(r), application.Filter{Query: q.Get("q"), PlanID: q.Get("planId"), Standing: q.Get("standing"), Page: page, Size: size})
	respond(w, 200, out, err)
}
func (h *Handler) GetPlatformOrganization(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Organization(r.Context(), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Handler) ListPlatformWhatsAppChannels(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Channels(r.Context())
	respond(w, 200, map[string]any{"items": out}, err)
}
func (h *Handler) OnboardTenant(w http.ResponseWriter, r *http.Request) {
	id := w.Header().Get("X-Request-ID")
	if id == "" {
		id = shared.NewCorrelationID()
		w.Header().Set("X-Request-ID", id)
	}
	ctx := observability.With(r.Context(), observability.Metadata{CorrelationID: id, Method: r.Method, Path: r.URL.Path})
	var in domain.TenantOnboardingInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "No pudimos leer los datos enviados. Revisa el formulario e inténtalo nuevamente.")
		return
	}
	out, err := h.service.Onboard(ctx, platformScope(r), in)
	respond(w, 201, out, err)
}
func (h *Handler) CreatePlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	var in domain.PlatformWhatsAppChannelInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_json", "El cuerpo de la solicitud no es válido.")
		return
	}
	out, err := h.service.CreateChannel(r.Context(), platformScope(r), in)
	respond(w, 201, out, err)
}
func (h *Handler) UpdatePlatformWhatsAppChannel(w http.ResponseWriter, r *http.Request) {
	var in domain.PlatformWhatsAppChannelInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_json", "El cuerpo de la solicitud no es válido.")
		return
	}
	out, err := h.service.UpdateChannel(r.Context(), platformScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
