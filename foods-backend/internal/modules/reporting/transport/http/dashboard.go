package http

import (
	"github.com/foods-platform/foods-backend/internal/modules/reporting/application"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service} }
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	s, _ := requestscope.From(r.Context())
	out, err := h.service.Dashboard(r.Context(), s, r.URL.Query().Get("period"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, out)
}
