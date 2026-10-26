package http

import (
	"encoding/json"
	"github.com/foods-platform/foods-backend/internal/modules/identity/application"
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"net/http"
	"strings"
)

type Handler struct {
	service      *application.Service
	secureCookie bool
}

func New(service *application.Service, secureCookie bool) *Handler {
	return &Handler{service, secureCookie}
}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		shared.Fail(w, 400, "invalid_request", "Revisa los datos enviados.")
		return
	}
	out, err := h.service.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	h.cookie(w, out.Token, out.ExpiresIn)
	shared.WriteJSON(w, 200, map[string]any{"user": map[string]any{"id": out.UserID, "name": out.Name, "platformAdmin": out.PlatformAdmin}, "expiresIn": out.ExpiresIn})
}
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := ""
	if c, err := r.Cookie("foods_session"); err == nil {
		token = c.Value
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		shared.WriteError(w, err)
		return
	}
	h.cookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) cookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: "foods_session", Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}
func (h *Handler) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie("foods_session"); err == nil {
			token = c.Value
		}
		if token == "" {
			token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		s, access, err := h.service.Authenticate(r.Context(), token)
		if err != nil {
			shared.WriteError(w, err)
			return
		}
		ctx := application.WithAccess(requestscope.With(r.Context(), s), s, access)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (h *Handler) RequirePermission(permission string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := shared.Scope(w, r)
		if !ok {
			return
		}
		if err := h.service.Authorize(r.Context(), s, permission); err != nil {
			shared.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
