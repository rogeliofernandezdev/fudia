package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (h *Handler) LinkWhatsApp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var in domain.WhatsAppLinkInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos de vinculación.")
		return
	}
	out, err := h.service.LinkWhatsApp(r.Context(), in)
	respond(w, 200, out, err)
}
func (h *Handler) WhatsAppRoutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.service.WhatsAppRoutes(r.Context())
	respond(w, 200, map[string]any{"items": out}, err)
}
func (h *Handler) WhatsAppHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.service.ChannelHistory(r.Context(), r.PathValue("id"))
	respond(w, 200, map[string]any{"items": out}, err)
}

func (h *Handler) RegisterLegacyRoute(w http.ResponseWriter, r *http.Request) {
	var in domain.WhatsAppRoute
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_route", "Revisa el canal.")
		return
	}
	err := h.service.RegisterLegacyRoute(r.Context(), in)
	respond(w, 200, map[string]bool{"registered": err == nil}, err)
}
