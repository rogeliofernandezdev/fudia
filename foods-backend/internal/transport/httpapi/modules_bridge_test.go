package httpapi

import (
	"context"

	auditpg "github.com/foods-platform/foods-backend/internal/modules/audit/infrastructure/postgres"
	subscriptionapp "github.com/foods-platform/foods-backend/internal/modules/subscriptions/application"
	subscriptiondomain "github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	subscriptionpg "github.com/foods-platform/foods-backend/internal/modules/subscriptions/infrastructure/postgres"
	"github.com/foods-platform/foods-backend/internal/platform/database"
)

const moduleAvailabilityReady = subscriptiondomain.ModuleAvailabilityReady
const moduleAvailabilityDevelopment = subscriptiondomain.ModuleAvailabilityDevelopment
const moduleAvailabilityPlanned = subscriptiondomain.ModuleAvailabilityPlanned

var mvpModuleKeys = subscriptiondomain.MvpModuleKeys
var moduleCatalog = subscriptiondomain.ModuleCatalog
var moduleAvailability = subscriptiondomain.ModuleAvailability
var moduleCanActivate = subscriptiondomain.ModuleCanActivate

func (a *integrationAPI) readOrganizationModules(ctx context.Context, org string) (map[string]bool, error) {
	return a.handlers.Modules.ActiveModules(ctx, org)
}
func (a *integrationAPI) activeModules(orgID string) map[string]bool {
	result, _ := a.readOrganizationModules(context.Background(), orgID)
	return result
}

// syncPlanModules applies a plan's modules to a company, as a plan or
// subscription change does.
func (a *integrationAPI) syncPlanModules(orgID string, keys []string) error {
	return subscriptionapp.New(subscriptionpg.New(a.db), database.NewUnitOfWork(a.db), auditpg.New(a.db)).SyncModules(context.Background(), orgID, keys)
}
