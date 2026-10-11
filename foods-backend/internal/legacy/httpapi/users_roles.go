package httpapi

import (
	"context"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"net/http"
)

type IdentityAdministrationHandler interface {
	GetContext(http.ResponseWriter, *http.Request)
	ListRoles(http.ResponseWriter, *http.Request)
	GetPermissionCatalog(http.ResponseWriter, *http.Request)
	CreateRole(http.ResponseWriter, *http.Request)
	UpdateRole(http.ResponseWriter, *http.Request)
	UpdateRoleStatus(http.ResponseWriter, *http.Request)
	ListUsers(http.ResponseWriter, *http.Request)
	CreateUser(http.ResponseWriter, *http.Request)
	UpdateUser(http.ResponseWriter, *http.Request)
	UpdateUserStatus(http.ResponseWriter, *http.Request)
	GetMyProfile(http.ResponseWriter, *http.Request)
	UpdateMyProfile(http.ResponseWriter, *http.Request)
	ListOrganizations(http.ResponseWriter, *http.Request)
	ListOrgLocations(http.ResponseWriter, *http.Request)
	SwitchContext(http.ResponseWriter, *http.Request)
	ListAvailableLocations(http.ResponseWriter, *http.Request)
	Catalog(context.Context, scope) (identitydomain.EffectiveRoleCatalog, error)
}

func (a *API) listRoles(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.ListRoles(w, r)
}
func (a *API) getPermissionCatalog(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.GetPermissionCatalog(w, r)
}
func (a *API) createRole(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.CreateRole(w, r)
}
func (a *API) updateRole(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.UpdateRole(w, r)
}
func (a *API) updateRoleStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.UpdateRoleStatus(w, r)
}
func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.ListUsers(w, r)
}
func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.CreateUser(w, r)
}
func (a *API) updateUser(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.UpdateUser(w, r)
}
func (a *API) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.UpdateUserStatus(w, r)
}
func (a *API) getMyProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.GetMyProfile(w, r)
}
func (a *API) updateMyProfile(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.UpdateMyProfile(w, r)
}
func (a *API) listOrganizations(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.ListOrganizations(w, r)
}
func (a *API) listOrgLocations(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.ListOrgLocations(w, r)
}
func (a *API) switchContext(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.SwitchContext(w, r)
}
func (a *API) listAvailableLocations(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.ListAvailableLocations(w, r)
}
func (a *API) loadRoleCatalog(ctx context.Context, s scope) (effectiveRoleCatalog, error) {
	return a.handlers.IdentityAdministration.Catalog(ctx, s)
}

func (a *API) saveRole(w http.ResponseWriter, r *http.Request, update bool) {
	if update {
		a.updateRole(w, r)
	} else {
		a.createRole(w, r)
	}
}

func (a *API) getContext(w http.ResponseWriter, r *http.Request) {
	a.handlers.IdentityAdministration.GetContext(w, r)
}
