package application

import (
	"context"
	"crypto/sha256"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

type UserFilter struct {
	Query, Status string
	Page, Size    int
}
type UserPage struct {
	Items    []domain.UserView `json:"items"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}
type ProfileCredentials struct{ Email, Hash string }
type UserState struct{ PlatformAdmin, Active bool }
type ContextDestination struct{ OrganizationName, LocationName, Country, Timezone string }
type ContextResult struct {
	Organization map[string]string `json:"organization"`
	Location     map[string]string `json:"location"`
}
type AdministrationRepository interface {
	Roles(context.Context, string) ([]domain.RoleView, error)
	SaveRole(context.Context, string, string, domain.RoleInput, []string, []string) (domain.RoleView, error)
	SetRoleActive(context.Context, string, string, bool) (bool, error)
	HasAdministrator(context.Context, string) (bool, error)
	PlatformAdmin(context.Context, string) (bool, error)
	CatalogAdministrator(context.Context, requestscope.Scope) (bool, error)
	Users(context.Context, string, UserFilter) (UserPage, error)
	LockUser(context.Context, string, string) (UserState, error)
	AssignmentValid(context.Context, string, domain.UserAssignment) (bool, error)
	SaveUser(context.Context, string, string, domain.UserInput, string) (string, error)
	ReplaceAssignments(context.Context, string, []domain.UserAssignment) error
	SetUserActive(context.Context, string, string, bool) error
	RevokeUserSessions(context.Context, string, []byte) error
	Profile(context.Context, requestscope.Scope) (domain.MyProfileView, error)
	LockProfile(context.Context, string) (ProfileCredentials, error)
	UpdateProfile(context.Context, string, string, string) error
	Organizations(context.Context) ([]domain.OrgSummary, error)
	Locations(context.Context, string) ([]domain.LocationSummary, error)
	AvailableLocations(context.Context, requestscope.Scope) ([]domain.LocationSummary, error)
	ContextDestination(context.Context, string, string) (ContextDestination, error)
	HasLocationRole(context.Context, string, string) (bool, error)
	SwitchSessionContext(context.Context, []byte, domain.SwitchContextInput) error
}
type Passwords interface {
	Matches(string, string) bool
	Hash(string) (string, error)
}

// OrganizationAccess is a collaboration port; subscription reads share the unit of work context.
type OrganizationAccess interface {
	ActiveModules(context.Context, string) (map[string]bool, error)
	EnsureCapacity(context.Context, string, string) error
}
type Administration struct {
	repo          AdministrationRepository
	work          unitofwork.Manager
	passwords     Passwords
	organizations OrganizationAccess
	audit         audit.Repository
}

func NewAdministration(repo AdministrationRepository, work unitofwork.Manager, passwords Passwords, organizations OrganizationAccess, audit audit.Repository) *Administration {
	return &Administration{repo, work, passwords, organizations, audit}
}
func failure(kind apperror.Kind, code, message string) error {
	return apperror.New(kind, code, message)
}
func (a *Administration) Catalog(ctx context.Context, s requestscope.Scope) (domain.EffectiveRoleCatalog, error) {
	platform, err := a.repo.CatalogAdministrator(ctx, s)
	if err != nil {
		return domain.EffectiveRoleCatalog{}, err
	}
	if platform {
		return domain.EffectiveRoleCatalog{Menus: domain.MenuAccessCatalog, Permissions: domain.PermissionCatalog, Unrestricted: true}, nil
	}
	modules, err := a.organizations.ActiveModules(ctx, s.OrganizationID)
	if err != nil {
		return domain.EffectiveRoleCatalog{}, err
	}
	return domain.RoleCatalogForModules(modules), nil
}
func (a *Administration) Roles(ctx context.Context, s requestscope.Scope) ([]domain.RoleView, error) {
	catalog, err := a.Catalog(ctx, s)
	if err != nil {
		return nil, failure(apperror.Unavailable, "role_catalog_unavailable", "No pudimos consultar los accesos del plan de la empresa.")
	}
	items, err := a.repo.Roles(ctx, s.OrganizationID)
	if err != nil {
		return nil, failure(apperror.Unavailable, "roles_unavailable", "No pudimos cargar los roles.")
	}
	for i := range items {
		items[i] = catalog.Project(items[i])
	}
	return items, nil
}
func (a *Administration) SaveRole(ctx context.Context, s requestscope.Scope, id string, in domain.RoleInput) (domain.RoleView, error) {
	var out domain.RoleView
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	menus, mok := domain.NormalizeRoleCatalogValues(in.MenuAccess, domain.MenuAccessCatalog)
	permissions, pok := domain.NormalizeRoleCatalogValues(in.Permissions, domain.PermissionCatalog)
	if in.Name == "" || len(in.Name) > 100 || len(in.Description) > 240 || !mok || !pok {
		return out, failure(apperror.Invalid, "invalid_role", "Ingresa un nombre válido y selecciona únicamente accesos y permisos disponibles.")
	}
	in.MenuAccess = menus
	in.Permissions = permissions
	catalog, err := a.Catalog(ctx, s)
	if err != nil {
		return out, failure(apperror.Unavailable, "role_catalog_unavailable", "No pudimos consultar los accesos del plan de la empresa.")
	}
	if _, ok := domain.NormalizeRoleCatalogValues(in.MenuAccess, catalog.Menus); !ok {
		return out, failure(apperror.Invalid, "role_outside_plan", "Selecciona únicamente opciones incluidas y activas en el plan de la empresa.")
	}
	if _, ok := domain.NormalizeRoleCatalogValues(in.Permissions, catalog.Permissions); !ok {
		return out, failure(apperror.Invalid, "role_outside_plan", "Selecciona únicamente acciones disponibles en el plan de la empresa.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, failure(apperror.Unavailable, "roles_unavailable", "No pudimos guardar el rol.")
	}
	defer tx.Rollback(ctx)
	out, err = a.repo.SaveRole(tx.Context(), s.OrganizationID, id, in, domain.RoleCatalogValues(catalog.Menus), domain.RoleCatalogValues(catalog.Permissions))
	if errors.Is(err, ErrNotFound) {
		return out, failure(apperror.Conflict, "protected_role", "El rol Administrador con acceso total no puede modificarse.")
	}
	if err != nil {
		return out, failure(apperror.Conflict, "role_conflict", "Ya existe un rol con ese nombre.")
	}
	if id != "" {
		if err = a.continuity(tx.Context(), s.OrganizationID, "roles_unavailable", "El cambio dejaría a la empresa sin un usuario con permisos para administrar accesos."); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, failure(apperror.Unavailable, "roles_unavailable", "No pudimos guardar el rol.")
	}
	action := "role.created"
	if id != "" {
		action = "role.updated"
	}
	_ = a.audit.Record(ctx, s, action, "role", out.ID)
	return catalog.Project(out), nil
}
func (a *Administration) continuity(ctx context.Context, org, code, message string) error {
	ok, err := a.repo.HasAdministrator(ctx, org)
	if err != nil {
		return failure(apperror.Unavailable, code, "No pudimos validar la continuidad administrativa.")
	}
	if !ok {
		return failure(apperror.Conflict, "last_administrator", message)
	}
	return nil
}
func (a *Administration) SetRoleActive(ctx context.Context, s requestscope.Scope, id string, active bool) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return failure(apperror.Unavailable, "roles_unavailable", "No pudimos actualizar el rol.")
	}
	defer tx.Rollback(ctx)
	changed, err := a.repo.SetRoleActive(tx.Context(), s.OrganizationID, id, active)
	if err != nil || !changed {
		return failure(apperror.Conflict, "protected_role", "El rol Administrador no puede desactivarse.")
	}
	if !active {
		if err = a.continuity(tx.Context(), s.OrganizationID, "roles_unavailable", "No puedes desactivar el último rol que permite administrar usuarios."); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return failure(apperror.Unavailable, "roles_unavailable", "No pudimos actualizar el rol.")
	}
	_ = a.audit.Record(ctx, s, "role.status_updated", "role", id)
	return nil
}
func (a *Administration) Users(ctx context.Context, s requestscope.Scope, f UserFilter) (UserPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	out, err := a.repo.Users(ctx, s.OrganizationID, f)
	if err != nil {
		return out, failure(apperror.Unavailable, "users_unavailable", "No pudimos cargar los usuarios.")
	}
	return out, nil
}
func (a *Administration) SaveUser(ctx context.Context, s requestscope.Scope, id string, in domain.UserInput) (string, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.FullName == "" || len(in.FullName) > 180 || !strings.Contains(in.Email, "@") || len(in.Email) > 180 || (id == "" && len(in.Password) < 8) || len(in.Assignments) == 0 {
		return "", failure(apperror.Invalid, "invalid_user", "Completa nombre, correo, rol, local y una contraseña de al menos 8 caracteres.")
	}
	if in.Password != "" && len(in.Password) < 8 {
		return "", failure(apperror.Invalid, "invalid_password", "La contraseña debe tener al menos 8 caracteres.")
	}
	seen := map[string]bool{}
	for _, v := range in.Assignments {
		k := strings.TrimSpace(v.RoleID) + ":" + strings.TrimSpace(v.LocationID)
		if strings.TrimSpace(v.RoleID) == "" || strings.TrimSpace(v.LocationID) == "" || seen[k] {
			return "", failure(apperror.Invalid, "invalid_assignment", "Cada acceso necesita un rol y local válidos, sin duplicados.")
		}
		seen[k] = true
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return "", failure(apperror.Unavailable, "user_unavailable", "No pudimos guardar el usuario.")
	}
	defer tx.Rollback(ctx)
	if id == "" {
		if err = a.organizations.EnsureCapacity(tx.Context(), s.OrganizationID, "users"); err != nil {
			return "", err
		}
	}
	actor, err := a.repo.PlatformAdmin(tx.Context(), s.UserID)
	if err != nil {
		return "", failure(apperror.Unavailable, "user_unavailable", "No pudimos validar al administrador actual.")
	}
	if id != "" {
		target, e := a.repo.LockUser(tx.Context(), s.OrganizationID, id)
		if errors.Is(e, ErrNotFound) {
			return "", failure(apperror.NotFound, "user_not_found", "El usuario no existe.")
		}
		if e != nil {
			return "", failure(apperror.Unavailable, "user_unavailable", "No pudimos validar el usuario.")
		}
		if target.PlatformAdmin && !actor {
			return "", failure(apperror.Forbidden, "protected_platform_admin", "Una cuenta de plataforma no puede modificarse desde la administración de una empresa.")
		}
	}
	for _, v := range in.Assignments {
		ok, e := a.repo.AssignmentValid(tx.Context(), s.OrganizationID, v)
		if e != nil {
			return "", failure(apperror.Unavailable, "user_unavailable", "No pudimos validar los accesos seleccionados.")
		}
		if !ok {
			return "", failure(apperror.Invalid, "invalid_assignment", "Uno de los roles o locales seleccionados ya no está disponible.")
		}
	}
	hash := ""
	if in.Password != "" {
		hash, err = a.passwords.Hash(in.Password)
		if err != nil {
			return "", failure(apperror.Internal, "user_unavailable", "No pudimos preparar la contraseña.")
		}
	}
	saved, err := a.repo.SaveUser(tx.Context(), s.OrganizationID, id, in, hash)
	if err != nil {
		return "", failure(apperror.Conflict, "user_conflict", "El correo ya está registrado.")
	}
	if err = a.repo.ReplaceAssignments(tx.Context(), saved, in.Assignments); err != nil {
		return "", failure(apperror.Invalid, "invalid_assignment", "No pudimos aplicar uno de los accesos seleccionados.")
	}
	if err = a.continuity(tx.Context(), s.OrganizationID, "user_unavailable", "El cambio dejaría a la empresa sin un usuario con permisos para administrar accesos."); err != nil {
		return "", err
	}
	if id != "" && in.Password != "" && saved != s.UserID {
		if err = a.repo.RevokeUserSessions(tx.Context(), saved, nil); err != nil {
			return "", failure(apperror.Unavailable, "user_unavailable", "No pudimos cerrar las sesiones anteriores del usuario.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return "", failure(apperror.Unavailable, "user_unavailable", "No pudimos guardar el usuario.")
	}
	action := "user.created"
	if id != "" {
		action = "user.updated"
	}
	_ = a.audit.Record(ctx, s, action, "user", saved)
	return saved, nil
}
func (a *Administration) SetUserActive(ctx context.Context, s requestscope.Scope, id string, active bool) error {
	if id == s.UserID {
		return failure(apperror.Conflict, "self_deactivation", "No puedes desactivar tu propia cuenta.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return failure(apperror.Unavailable, "user_unavailable", "No pudimos actualizar el usuario.")
	}
	defer tx.Rollback(ctx)
	actor, err := a.repo.PlatformAdmin(tx.Context(), s.UserID)
	if err != nil {
		return failure(apperror.Unavailable, "user_unavailable", "No pudimos validar al administrador actual.")
	}
	target, err := a.repo.LockUser(tx.Context(), s.OrganizationID, id)
	if errors.Is(err, ErrNotFound) {
		return failure(apperror.NotFound, "user_not_found", "El usuario no existe.")
	}
	if err != nil {
		return failure(apperror.Unavailable, "user_unavailable", "No pudimos validar el usuario.")
	}
	if target.PlatformAdmin && !actor {
		return failure(apperror.Forbidden, "protected_platform_admin", "Una cuenta de plataforma no puede modificarse desde la administración de una empresa.")
	}
	if active && !target.Active && !target.PlatformAdmin {
		if err = a.organizations.EnsureCapacity(tx.Context(), s.OrganizationID, "users"); err != nil {
			return err
		}
	}
	if err = a.repo.SetUserActive(tx.Context(), s.OrganizationID, id, active); err != nil {
		return failure(apperror.Unavailable, "user_unavailable", "No pudimos actualizar el usuario.")
	}
	if !active {
		if err = a.continuity(tx.Context(), s.OrganizationID, "user_unavailable", "No puedes desactivar al último usuario que administra accesos."); err != nil {
			return err
		}
		if err = a.repo.RevokeUserSessions(tx.Context(), id, nil); err != nil {
			return failure(apperror.Unavailable, "user_unavailable", "No pudimos cerrar las sesiones del usuario.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return failure(apperror.Unavailable, "user_unavailable", "No pudimos actualizar el usuario.")
	}
	_ = a.audit.Record(ctx, s, "user.status_updated", "user", id)
	return nil
}
func (a *Administration) Profile(ctx context.Context, s requestscope.Scope) (domain.MyProfileView, error) {
	out, err := a.repo.Profile(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return out, failure(apperror.NotFound, "profile_not_found", "Tu cuenta ya no está disponible.")
	}
	if err != nil {
		return out, failure(apperror.Unavailable, "profile_unavailable", "No pudimos cargar tu perfil.")
	}
	if out.PlatformAdmin {
		out.RoleNames = []string{"Administrador de plataforma"}
		out.Permissions = []string{"*"}
	}
	if out.RoleNames == nil {
		out.RoleNames = []string{}
	}
	if out.Permissions == nil {
		out.Permissions = []string{}
	}
	return out, nil
}
func (a *Administration) UpdateProfile(ctx context.Context, s requestscope.Scope, in domain.MyProfileInput, token string) (domain.MyProfileView, error) {
	var out domain.MyProfileView
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" || len(in.FullName) > 180 {
		return out, failure(apperror.Invalid, "invalid_profile", "Ingresa un nombre válido.")
	}
	if in.NewPassword != "" && len(in.NewPassword) < 8 {
		return out, failure(apperror.Invalid, "invalid_password", "La nueva contraseña debe tener al menos 8 caracteres.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, failure(apperror.Unavailable, "profile_unavailable", "No pudimos actualizar tu perfil.")
	}
	defer tx.Rollback(ctx)
	current, err := a.repo.LockProfile(tx.Context(), s.UserID)
	if errors.Is(err, ErrNotFound) {
		return out, failure(apperror.NotFound, "profile_not_found", "Tu cuenta ya no está disponible.")
	}
	if err != nil {
		return out, failure(apperror.Unavailable, "profile_unavailable", "No pudimos validar tu cuenta.")
	}
	hash := current.Hash
	if in.NewPassword != "" {
		if !a.passwords.Matches(hash, in.CurrentPassword) {
			return out, failure(apperror.Conflict, "current_password_invalid", "La contraseña actual no es correcta.")
		}
		hash, err = a.passwords.Hash(in.NewPassword)
		if err != nil {
			return out, failure(apperror.Internal, "profile_unavailable", "No pudimos preparar la nueva contraseña.")
		}
	}
	if err = a.repo.UpdateProfile(tx.Context(), s.UserID, in.FullName, hash); err != nil {
		return out, failure(apperror.Unavailable, "profile_unavailable", "No pudimos actualizar tu perfil.")
	}
	if in.NewPassword != "" && token != "" {
		sum := sha256.Sum256([]byte(token))
		if err = a.repo.RevokeUserSessions(tx.Context(), s.UserID, sum[:]); err != nil {
			return out, failure(apperror.Unavailable, "profile_unavailable", "No pudimos cerrar las otras sesiones.")
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, failure(apperror.Unavailable, "profile_unavailable", "No pudimos actualizar tu perfil.")
	}
	_ = a.audit.Record(ctx, s, "user.profile_updated", "user", s.UserID)
	out, err = a.Profile(ctx, s)
	if err != nil {
		return domain.MyProfileView{FullName: in.FullName, Email: current.Email, RoleNames: []string{}, Permissions: []string{}}, nil
	}
	return out, nil
}
func (a *Administration) Organizations(ctx context.Context) ([]domain.OrgSummary, error) {
	items, err := a.repo.Organizations(ctx)
	if err != nil {
		return nil, failure(apperror.Unavailable, "orgs_unavailable", "No pudimos cargar las empresas.")
	}
	return items, nil
}
func (a *Administration) Locations(ctx context.Context, id string) ([]domain.LocationSummary, error) {
	if id == "" {
		return nil, failure(apperror.Invalid, "invalid_request", "ID de empresa requerido.")
	}
	items, err := a.repo.Locations(ctx, id)
	if err != nil {
		return nil, failure(apperror.Unavailable, "locations_unavailable", "No pudimos cargar los locales.")
	}
	return items, nil
}
func (a *Administration) AvailableLocations(ctx context.Context, s requestscope.Scope) ([]domain.LocationSummary, error) {
	items, err := a.repo.AvailableLocations(ctx, s)
	if err != nil {
		return nil, failure(apperror.Unavailable, "locations_unavailable", "No pudimos cargar los locales.")
	}
	return items, nil
}
func (a *Administration) SwitchContext(ctx context.Context, s requestscope.Scope, in domain.SwitchContextInput, token string) (ContextResult, error) {
	var out ContextResult
	if in.OrganizationID == "" || in.LocationID == "" {
		return out, failure(apperror.Invalid, "invalid_request", "Empresa y local son obligatorios.")
	}
	target, err := a.repo.ContextDestination(ctx, in.OrganizationID, in.LocationID)
	if err != nil {
		return out, failure(apperror.NotFound, "context_not_found", "La empresa o local no existe o está inactivo.")
	}
	admin, err := a.repo.PlatformAdmin(ctx, s.UserID)
	if err != nil {
		return out, failure(apperror.Unavailable, "session_error", "No pudimos cambiar el contexto.")
	}
	if !admin {
		if in.OrganizationID != s.OrganizationID {
			return out, failure(apperror.Forbidden, "forbidden", "No puedes operar en otra empresa.")
		}
		ok, e := a.repo.HasLocationRole(ctx, s.UserID, in.LocationID)
		if e != nil {
			return out, failure(apperror.Unavailable, "session_error", "No pudimos cambiar el contexto.")
		}
		if !ok {
			return out, failure(apperror.Forbidden, "forbidden", "No tienes un rol asignado en ese local.")
		}
	}
	if token == "" {
		return out, failure(apperror.Unauthenticated, "session_required", "No hay sesión activa.")
	}
	sum := sha256.Sum256([]byte(token))
	if err = a.repo.SwitchSessionContext(ctx, sum[:], in); err != nil {
		return out, failure(apperror.Unavailable, "session_error", "No pudimos cambiar el contexto.")
	}
	_ = a.audit.Record(ctx, s, "context.switched", "organization", in.OrganizationID)
	return ContextResult{map[string]string{"id": in.OrganizationID, "name": target.OrganizationName}, map[string]string{"id": in.LocationID, "name": target.LocationName, "country": target.Country, "timezone": target.Timezone}}, nil
}
