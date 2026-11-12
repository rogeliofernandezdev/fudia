package http

import "net/http"

func (h *Handler) GeographicAreas(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.GeographicAreas(r.Context(), r.URL.Query().Get("parentCode"))
	respond(w, http.StatusOK, map[string]any{"items": items}, err)
}
