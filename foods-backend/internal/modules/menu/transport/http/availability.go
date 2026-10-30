package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
	"regexp"
	"strings"
)

var availabilityProductID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (h *Catalog) ListProductAvailability(w http.ResponseWriter, r *http.Request) {
	filter := catalogFilter(r)
	if raw, present := r.URL.Query()["productIds"]; present {
		if len(raw) != 1 || raw[0] == "" {
			shared.Fail(w, 400, "invalid_product_ids", "Selecciona entre 1 y 100 productos válidos.")
			return
		}
		filter.ProductIDs = strings.Split(raw[0], ",")
		if len(filter.ProductIDs) > 100 {
			shared.Fail(w, 400, "invalid_product_ids", "Selecciona entre 1 y 100 productos válidos.")
			return
		}
		for _, id := range filter.ProductIDs {
			if !availabilityProductID.MatchString(id) {
				shared.Fail(w, 400, "invalid_product_ids", "Selecciona entre 1 y 100 productos válidos.")
				return
			}
		}
	}
	out, err := h.service.Availability(r.Context(), menuScope(r), filter, r.URL.Query().Get("excludeCombos") == "true")
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
