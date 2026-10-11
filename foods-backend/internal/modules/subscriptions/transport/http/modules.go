package http

import (
	"context"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Modules struct{ service *application.Modules }

func NewModules(service *application.Modules) *Modules { return &Modules{service} }
func (h *Modules) ActiveModules(ctx context.Context, org string) (map[string]bool, error) {
	return h.service.ActiveModules(ctx, org)
}
func (h *Modules) ListModules(w http.ResponseWriter, r *http.Request) {
	s, _ := requestscope.From(r.Context())
	out, err := h.service.List(r.Context(), s.OrganizationID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]any{"modules": out})
}
func (h *Modules) ToggleModule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key    string `json:"key"`
		Active bool   `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	s, _ := requestscope.From(r.Context())
	if err := h.service.SetActive(r.Context(), s, in.Key, in.Active); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}
