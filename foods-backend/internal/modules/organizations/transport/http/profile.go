package http

import (
	"encoding/json"
	menucontracts "github.com/foods-platform/foods-backend/internal/modules/menu/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

func (h *Handler) WithProfiles(profiles *application.RestaurantProfiles) *Handler {
	h.profiles = profiles
	return h
}
func (h *Handler) profileScope(w http.ResponseWriter, r *http.Request) (requestscope.Scope, bool) {
	w.Header().Set("Cache-Control", "private, no-store")
	if h.profiles == nil {
		shared.Fail(w, 503, "profile_unavailable", "La identidad del restaurante no está configurada.")
		return requestscope.Scope{}, false
	}
	s, err := h.profiles.Target(r.Context(), orgScope(r), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return s, false
	}
	return s, true
}
func (h *Handler) RestaurantProfile(w http.ResponseWriter, r *http.Request) {
	s, ok := h.profileScope(w, r)
	if !ok {
		return
	}
	out, err := h.profiles.Read(r.Context(), s)
	respond(w, 200, out, err)
}
func (h *Handler) SaveRestaurantProfile(w http.ResponseWriter, r *http.Request) {
	s, ok := h.profileScope(w, r)
	if !ok {
		return
	}
	var in application.RestaurantProfileInput
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_profile", "Revisa los datos del restaurante.")
		return
	}
	respond(w, 204, nil, h.profiles.Save(r.Context(), s, in))
}
func (h *Handler) AuthorizeRestaurantImage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.profileScope(w, r)
	if !ok {
		return
	}
	var in menucontracts.MediaMetadata
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_image", "Revisa la imagen seleccionada.")
		return
	}
	out, err := h.profiles.Authorize(r.Context(), s, in)
	respond(w, 201, out, err)
}
func (h *Handler) CompleteRestaurantImage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.profileScope(w, r)
	if !ok {
		return
	}
	out, err := h.profiles.Complete(r.Context(), s, r.PathValue("imageId"))
	respond(w, 200, out, err)
}
func (h *Handler) RestaurantImage(w http.ResponseWriter, r *http.Request) {
	s, ok := h.profileScope(w, r)
	if !ok {
		return
	}
	out, err := h.profiles.Image(r.Context(), s, r.PathValue("imageId"))
	respond(w, 200, out, err)
}
func (h *Handler) RestaurantBranding(w http.ResponseWriter, r *http.Request) {
	s, ok := h.profileScope(w, r)
	if !ok {
		return
	}
	out, err := h.profiles.ReportBranding(r.Context(), s)
	respond(w, 200, out, err)
}
