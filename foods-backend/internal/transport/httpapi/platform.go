package httpapi

import (
	shared "github.com/foods-platform/foods-backend/internal/platform/httpapi"
	"net/http"
)

func (a *API) requirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := r.Context().Value(scopeKey{}).(scope)
		if err := a.handlers.Platform.Authorize(r.Context(), s.UserID); err != nil {
			shared.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *API) onboardTenant(w http.ResponseWriter, r *http.Request) {
	a.handlers.Platform.OnboardTenant(w, r)
}
