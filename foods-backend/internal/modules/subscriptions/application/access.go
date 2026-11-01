package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

type Capacity struct {
	Maximum *int
	Used    int
}
type AccessRepository interface {
	ModuleSwitches(context.Context, string) (map[string]domain.ModuleSwitch, error)
	LockCapacity(context.Context, string, string) (Capacity, error)
}
type Access struct{ repo AccessRepository }

func NewAccess(repo AccessRepository) *Access { return &Access{repo} }

// ModuleEntitled reports whether the company can use the module, with the same
// rule as the session (domain.ModuleEnabled), so a platform pilot of a module
// in development shows its screens and passes its backend checks alike.
func (a *Access) ModuleEntitled(ctx context.Context, org, key string) (bool, error) {
	modules, err := a.ActiveModules(ctx, org)
	if err != nil {
		return false, err
	}
	return modules[key], nil
}
func (a *Access) ActiveModules(ctx context.Context, org string) (map[string]bool, error) {
	switches, err := a.repo.ModuleSwitches(ctx, org)
	if err != nil {
		return nil, err
	}
	modules := make(map[string]bool, len(switches))
	for key, state := range switches {
		modules[key] = domain.ModuleEnabled(key, state)
	}
	return modules, nil
}
func (a *Access) EnsureCapacity(ctx context.Context, org, resource string) error {
	capacity, err := a.repo.LockCapacity(ctx, org, resource)
	if err != nil {
		return apperror.New(apperror.Unavailable, "subscription_unavailable", "No pudimos validar los límites del plan.")
	}
	if capacity.Maximum != nil && capacity.Used >= *capacity.Maximum {
		message := "El plan contratado alcanzó el máximo de usuarios."
		if resource == "locations" {
			message = "El plan contratado alcanzó el máximo de locales."
		}
		return apperror.New(apperror.Conflict, "plan_limit_reached", message)
	}
	return nil
}
