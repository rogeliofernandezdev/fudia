package httpapi

import (
	"context"
	"crypto/sha256"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	platformpg "github.com/foods-platform/foods-backend/internal/modules/platformadmin/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type roleView = identitydomain.RoleView
type userView = identitydomain.UserView

var permissionCatalog = identitydomain.PermissionCatalog
var menuAccessCatalog = identitydomain.MenuAccessCatalog

type myProfileView = identitydomain.MyProfileView
type locationSummary = identitydomain.LocationSummary
type switchContextInput = identitydomain.SwitchContextInput
type effectiveRoleCatalog = identitydomain.EffectiveRoleCatalog

var defaultOrganizationRoles = identitydomain.DefaultOrganizationRoles
var permissionAvailableForModules = identitydomain.PermissionAvailableForModules
var roleCatalogForModules = identitydomain.RoleCatalogForModules
var roleCatalogValues = identitydomain.RoleCatalogValues

func seedOrganizationRoles(ctx context.Context, tx pgx.Tx, organizationID string) (string, error) {
	ctx = database.WithTransaction(ctx, nil, tx)
	repo := platformpg.New(nil)
	administratorID := ""
	for _, seed := range identitydomain.DefaultOrganizationRoles {
		id, err := repo.SeedRole(ctx, organizationID, seed)
		if err != nil {
			return "", err
		}
		if seed.SystemKey == "administrator" {
			administratorID = id
		}
	}
	return administratorID, nil
}
func sha256Sum(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

var rolePermissionModules = identitydomain.RolePermissionModules
