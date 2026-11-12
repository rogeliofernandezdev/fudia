package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (h *Handler) Restaurants(w http.ResponseWriter, r *http.Request) {
	var in domain.RestaurantSearchInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_customer_identity", "Revisa los datos del cliente.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.service.Restaurants(r.Context(), in)
	respond(w, 200, out, err)
}
func (h *Handler) RestaurantContext(w http.ResponseWriter, r *http.Request) {
	var in domain.RestaurantSelectionInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_restaurant", "Selecciona un restaurante de la lista.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out, err := h.service.RestaurantContext(r.Context(), in)
	respond(w, 200, out, err)
}
