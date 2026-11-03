package http

import (
	"net/http"
	"strconv"

	"github.com/foods-platform/foods-backend/internal/modules/notifications/application"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type Handler struct{ service *application.Notices }

func New(service *application.Notices) *Handler { return &Handler{service} }

func noticeScope(r *http.Request) requestscope.Scope {
	s, _ := requestscope.From(r.Context())
	return s
}

func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Feed(r.Context(), noticeScope(r))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkRead(r.Context(), noticeScope(r), r.PathValue("id")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkAllRead(r.Context(), noticeScope(r)); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClaimCustomer entrega a Concierge (autenticado con su clave de servicio) los
// avisos al cliente pendientes de enviar.
func (h *Handler) ClaimCustomer(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := h.service.ClaimCustomer(r.Context(), limit)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) MarkCustomerDispatched(w http.ResponseWriter, r *http.Request) {
	if err := h.service.MarkCustomerDispatched(r.Context(), r.PathValue("id")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
