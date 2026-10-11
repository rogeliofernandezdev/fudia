package httpapi

import (
	"context"

	"net/http"
)

func (a *integrationAPI) loadRoleCatalog(ctx context.Context, s scope) (effectiveRoleCatalog, error) {
	return a.handlers.IdentityAdministration.Catalog(ctx, s)
}
func (a *integrationAPI) saveRole(w http.ResponseWriter, r *http.Request, update bool) {
	if update {
		a.updateRole(w, r)
	} else {
		a.createRole(w, r)
	}
}
