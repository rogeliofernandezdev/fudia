package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (h *Catalog) ListProductAvailability(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Availability(r.Context(), menuScope(r), catalogFilter(r), r.URL.Query().Get("excludeCombos") == "true")
	respond(w, 200, out, err)
}
func (h *Catalog) ListProductAvailabilityHistory(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.AvailabilityHistory(r.Context(), menuScope(r), r.PathValue("productId"), catalogFilter(r))
	respond(w, 200, out, err)
}
func (h *Catalog) UpdateProductAvailability(w http.ResponseWriter, r *http.Request) {
	var in domain.AvailabilityInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_availability", "La disponibilidad enviada no es válida.")
		return
	}
	err := h.service.UpdateAvailability(r.Context(), menuScope(r), r.PathValue("productId"), in)
	respond(w, 204, nil, err)
}
