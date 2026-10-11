package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/menu/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (h *Catalog) ListRecipes(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Recipes(r.Context(), menuScope(r), catalogFilter(r))
	respond(w, 200, out, err)
}
func (h *Catalog) GetRecipe(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Recipe(r.Context(), menuScope(r), r.PathValue("productId"))
	respond(w, 200, out, err)
}
func (h *Catalog) SaveRecipe(w http.ResponseWriter, r *http.Request) {
	var in domain.RecipeInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_recipe", "Indica rendimiento e insumos válidos.")
		return
	}
	out, err := h.service.SaveRecipe(r.Context(), menuScope(r), r.PathValue("productId"), in)
	respond(w, 200, out, err)
}
