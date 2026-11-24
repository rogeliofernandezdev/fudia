package http

import (
	"encoding/json"
	"net/http"

	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
)

func (h *Stock) GetInventoryItem(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.ItemDetail(r.Context(), stockScope(r), r.PathValue("id"))
	stockResponse(w, 200, out, err)
}
func (h *Stock) PreviewInventoryUnitChange(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.PreviewUnitChange(r.Context(), stockScope(r), r.PathValue("id"), r.URL.Query().Get("unit"))
	stockResponse(w, 200, out, err)
}
func (h *Stock) UpdateInventoryItemStatus(w http.ResponseWriter, r *http.Request) {
	var in domain.ItemStatusInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_status", "Indica si el artículo queda activo.")
		return
	}
	err := h.service.SetItemStatus(r.Context(), stockScope(r), r.PathValue("id"), in)
	stockResponse(w, 204, nil, err)
}
func (h *Stock) RemoveInventoryPresentation(w http.ResponseWriter, r *http.Request) {
	err := h.service.RemovePresentation(r.Context(), stockScope(r), r.PathValue("id"), r.PathValue("presentationId"))
	stockResponse(w, 204, nil, err)
}
func (h *Stock) ListAssignableItems(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.AssignableItems(r.Context(), stockScope(r), r.URL.Query().Get("q"))
	stockResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Stock) AssignInventoryItem(w http.ResponseWriter, r *http.Request) {
	err := h.service.AssignItem(r.Context(), stockScope(r), r.PathValue("id"))
	stockResponse(w, 204, nil, err)
}
