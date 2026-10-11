package httpapi

import (
	"context"
	subscriptiondomain "github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	"github.com/jackc/pgx/v5"
	"net/http"
)

type moduleDef = subscriptiondomain.ModuleDef

const moduleAvailabilityReady = subscriptiondomain.ModuleAvailabilityReady
const moduleAvailabilityDevelopment = subscriptiondomain.ModuleAvailabilityDevelopment
const moduleAvailabilityPlanned = subscriptiondomain.ModuleAvailabilityPlanned

var mvpModuleKeys = subscriptiondomain.MvpModuleKeys
var developmentModuleKeys = subscriptiondomain.DevelopmentModuleKeys
var moduleCatalog = subscriptiondomain.ModuleCatalog
var moduleAvailability = subscriptiondomain.ModuleAvailability
var moduleCanActivate = subscriptiondomain.ModuleCanActivate
var moduleCategories = subscriptiondomain.ModuleCategories

type ModulesHandler interface {
	ListModules(http.ResponseWriter, *http.Request)
	ToggleModule(http.ResponseWriter, *http.Request)
	ActiveModules(context.Context, string) (map[string]bool, error)
}

func (a *API) listModules(w http.ResponseWriter, r *http.Request) {
	a.handlers.Modules.ListModules(w, r)
}
func (a *API) toggleModule(w http.ResponseWriter, r *http.Request) {
	a.handlers.Modules.ToggleModule(w, r)
}
func (a *API) readOrganizationModules(ctx context.Context, org string) (map[string]bool, error) {
	return a.handlers.Modules.ActiveModules(ctx, org)
}
func (a *API) activeModules(orgID string) map[string]bool {
	result, _ := a.readOrganizationModules(context.Background(), orgID)
	return result
}
func seedOrganizationModules(ctx context.Context, tx pgx.Tx, organizationID string) error {
	keys := make([]string, 0, len(mvpModuleKeys))
	for key := range mvpModuleKeys {
		keys = append(keys, key)
	}
	return syncOrganizationModulesForPlan(ctx, tx, organizationID, keys)
}
func syncOrganizationModulesForPlan(ctx context.Context, tx pgx.Tx, organizationID string, moduleKeys []string) error {
	allowed := map[string]bool{}
	for _, key := range moduleKeys {
		allowed[key] = true
	}
	for _, module := range moduleCatalog {
		active := allowed[module.Key] && moduleCanActivate(module.Key)
		_, err := tx.Exec(ctx, `
			INSERT INTO organization_modules(organization_id,module_key,active)
			VALUES($1,$2,$3)
			ON CONFLICT(organization_id,module_key)
			DO UPDATE SET active=EXCLUDED.active,updated_at=now()
		`, organizationID, module.Key, active)
		if err != nil {
			return err
		}
	}
	return nil
}
