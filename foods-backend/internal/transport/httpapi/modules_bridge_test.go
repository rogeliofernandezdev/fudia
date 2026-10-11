package httpapi

import (
	"context"
	subscriptiondomain "github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
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
