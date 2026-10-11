// Package contracts publishes role templates needed by tenant provisioning.
package contracts

import "github.com/foods-platform/foods-backend/internal/modules/identity/domain"

type OrganizationRoleSeed = domain.OrganizationRoleSeed

var DefaultOrganizationRoles = domain.DefaultOrganizationRoles
