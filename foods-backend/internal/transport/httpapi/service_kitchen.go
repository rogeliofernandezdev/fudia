package httpapi

import (
	"net/http"
	"strings"
)

func (a *API) requirePreparationPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		permission := "kitchen.manage"
		if strings.HasSuffix(r.PathValue("id"), "~bar") {
			permission = "bar.manage"
		}
		a.requirePermission(permission, next).ServeHTTP(w, r)
	})
}
