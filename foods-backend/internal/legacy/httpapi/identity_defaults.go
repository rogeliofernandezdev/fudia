package httpapi

import (
	"context"
	"crypto/sha256"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	"github.com/jackc/pgx/v5"
)

type roleView = identitydomain.RoleView
type roleInput = identitydomain.RoleInput
type userAssignment = identitydomain.UserAssignment
type userView = identitydomain.UserView
type userInput = identitydomain.UserInput

var permissionCatalog = identitydomain.PermissionCatalog
var menuAccessCatalog = identitydomain.MenuAccessCatalog

type myProfilePlanView = identitydomain.MyProfilePlanView
type myProfileView = identitydomain.MyProfileView
type myProfileInput = identitydomain.MyProfileInput
type orgSummary = identitydomain.OrgSummary
type locationSummary = identitydomain.LocationSummary
type switchContextInput = identitydomain.SwitchContextInput
type effectiveRoleCatalog = identitydomain.EffectiveRoleCatalog
type organizationRoleSeed = identitydomain.OrganizationRoleSeed

var companyAdministratorMenuAccess = identitydomain.CompanyAdministratorMenuAccess
var companyAdministratorPermissions = identitydomain.CompanyAdministratorPermissions
var defaultOrganizationRoles = identitydomain.DefaultOrganizationRoles
var normalizeRoleCatalogValues = identitydomain.NormalizeRoleCatalogValues
var permissionAvailableForModules = identitydomain.PermissionAvailableForModules
var filterRoleCatalog = identitydomain.FilterRoleCatalog
var roleCatalogForModules = identitydomain.RoleCatalogForModules
var roleCatalogValues = identitydomain.RoleCatalogValues
var effectiveRoleValues = identitydomain.EffectiveRoleValues

func seedOrganizationRoles(ctx context.Context, tx pgx.Tx, organizationID string) (string, error) {
	var administratorID string
	for _, seed := range defaultOrganizationRoles {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO roles(organization_id,name,system_key,description,menu_access,permissions,active)
			VALUES($1,$2,$3,$4,$5,$6,true)
			ON CONFLICT(organization_id,name) DO NOTHING
			RETURNING id
		`, organizationID, seed.Name, seed.SystemKey, seed.Description, seed.MenuAccess, seed.Permissions).Scan(&id)
		if err != nil && err != pgx.ErrNoRows {
			return "", err
		}
		if err == pgx.ErrNoRows {
			if err = tx.QueryRow(ctx, `SELECT id FROM roles WHERE organization_id=$1 AND name=$2`, organizationID, seed.Name).Scan(&id); err != nil {
				return "", err
			}
		}
		if seed.SystemKey == "administrator" {
			administratorID = id
		}
	}
	return administratorID, nil
}

type identityQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func tenantHasAdministrator(ctx context.Context, q identityQueryRower, organizationID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users u
			JOIN user_roles ur ON ur.user_id=u.id
			JOIN roles r ON r.id=ur.role_id AND r.organization_id=u.organization_id
			JOIN locations l ON l.id=ur.location_id AND l.organization_id=u.organization_id
			WHERE u.organization_id=$1
			  AND u.active
			  AND NOT u.platform_admin
			  AND r.active
			  AND l.active
			  AND r.permissions @> ARRAY['users.manage']::text[]
		)
	`, organizationID).Scan(&ok)
	return ok, err
}
func actorIsPlatformAdmin(ctx context.Context, q identityQueryRower, userID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT platform_admin FROM users WHERE id=$1 AND active`, userID).Scan(&ok)
	return ok, err
}
func sha256Sum(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

var rolePermissionModules = identitydomain.RolePermissionModules
