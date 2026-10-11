package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/application"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

type UnitHandler struct{ service *application.UnitService }

func NewUnits(service *application.UnitService) *UnitHandler { return &UnitHandler{service} }
func (h *UnitHandler) List(w http.ResponseWriter, r *http.Request) {
	s, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	items, err := h.service.List(r.Context(), s.OrganizationID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]any{"items": items})
}
func (h *UnitHandler) Create(w http.ResponseWriter, r *http.Request) {
	s, ok := shared.Scope(w, r)
	if !ok {
		return
	}
	var in domain.UnitInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_inventory_unit", "Revisa los datos de la unidad.")
		return
	}
	out, err := h.service.Create(r.Context(), s, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, out)
}
