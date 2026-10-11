package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (h *Catalog) GetProductModifiers(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Modifiers(r.Context(), menuScope(r), r.PathValue("id"))
	respond(w, 200, out, err)
}
func (h *Catalog) SaveProductModifiers(w http.ResponseWriter, r *http.Request) {
	var in domain.ModifierConfigInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_modifiers", "Revisa la configuración de modificadores.")
		return
	}
	out, err := h.service.SaveModifiers(r.Context(), menuScope(r), r.PathValue("id"), in)
	respond(w, 200, out, err)
}
