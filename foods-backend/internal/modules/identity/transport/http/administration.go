package http

import (
	"context"
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/identity/application"
	"github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
)

type Administration struct{ service *application.Administration }

func NewAdministration(service *application.Administration) *Administration {
	return &Administration{service}
}
func adminScope(r *http.Request) requestscope.Scope { s, _ := requestscope.From(r.Context()); return s }
func adminToken(r *http.Request) string {
	c, err := r.Cookie("foods_session")
	if err != nil {
		return ""
	}
	return c.Value
}
func adminResponse(w http.ResponseWriter, status int, out any, err error) {
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if status == 204 {
		w.WriteHeader(status)
		return
	}
	shared.WriteJSON(w, status, out)
}
func (h *Administration) Catalog(ctx context.Context, s requestscope.Scope) (domain.EffectiveRoleCatalog, error) {
	return h.service.Catalog(ctx, s)
}
func (h *Administration) ListRoles(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Roles(r.Context(), adminScope(r))
	adminResponse(w, 200, map[string]any{"items": out, "total": len(out)}, err)
}
func (h *Administration) GetPermissionCatalog(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Catalog(r.Context(), adminScope(r))
	if err != nil {
		shared.Fail(w, 503, "role_catalog_unavailable", "No pudimos consultar los accesos del plan de la empresa.")
		return
	}
	shared.WriteJSON(w, 200, map[string]any{"groups": out.Permissions, "menuGroups": out.Menus})
}
func (h *Administration) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, size := shared.PageParams(r)
	out, err := h.service.Users(r.Context(), adminScope(r), application.UserFilter{Query: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"), Page: page, Size: size})
	adminResponse(w, 200, out, err)
}
func (h *Administration) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Profile(r.Context(), adminScope(r))
	adminResponse(w, 200, out, err)
}
func (h *Administration) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Organizations(r.Context())
	adminResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Administration) ListOrgLocations(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Locations(r.Context(), r.PathValue("id"))
	adminResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Administration) ListAvailableLocations(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.AvailableLocations(r.Context(), adminScope(r))
	adminResponse(w, 200, map[string]any{"items": out}, err)
}
func (h *Administration) CreateRole(w http.ResponseWriter, r *http.Request) {
	var in domain.RoleInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_role", "Revisa los datos del rol.")
		return
	}
	out, err := h.service.SaveRole(r.Context(), adminScope(r), "", in)
	adminResponse(w, 201, out, err)
}
func (h *Administration) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var in domain.RoleInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_role", "Revisa los datos del rol.")
		return
	}
	out, err := h.service.SaveRole(r.Context(), adminScope(r), r.PathValue("id"), in)
	adminResponse(w, 200, out, err)
}
func (h *Administration) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in domain.UserInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_user", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SaveUser(r.Context(), adminScope(r), "", in)
	adminResponse(w, 201, map[string]string{"id": out}, err)
}
func (h *Administration) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var in domain.UserInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_user", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SaveUser(r.Context(), adminScope(r), r.PathValue("id"), in)
	adminResponse(w, 200, map[string]string{"id": out}, err)
}
func (h *Administration) UpdateRoleStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "Estado inválido.")
		return
	}
	err := h.service.SetRoleActive(r.Context(), adminScope(r), r.PathValue("id"), in.Active)
	adminResponse(w, 204, nil, err)
}
func (h *Administration) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("id") == adminScope(r).UserID {
		shared.Fail(w, 409, "self_deactivation", "No puedes desactivar tu propia cuenta.")
		return
	}
	var in struct {
		Active bool `json:"active"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_status", "Estado inválido.")
		return
	}
	err := h.service.SetUserActive(r.Context(), adminScope(r), r.PathValue("id"), in.Active)
	adminResponse(w, 204, nil, err)
}
func (h *Administration) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	var in domain.MyProfileInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_profile", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.UpdateProfile(r.Context(), adminScope(r), in, adminToken(r))
	adminResponse(w, 200, out, err)
}
func (h *Administration) SwitchContext(w http.ResponseWriter, r *http.Request) {
	var in domain.SwitchContextInput
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.SwitchContext(r.Context(), adminScope(r), in, adminToken(r))
	adminResponse(w, 200, out, err)
}
func (h *Administration) GetContext(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.Context(r.Context(), adminScope(r))
	adminResponse(w, 200, out, err)
}
