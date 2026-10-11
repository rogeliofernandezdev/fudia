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
	ActiveModules(context.Context, string) (map[string]bool, error)
	LockCapacity(context.Context, string, string) (Capacity, error)
}
type Access struct{ repo AccessRepository }

func NewAccess(repo AccessRepository) *Access { return &Access{repo} }
func (a *Access) ActiveModules(ctx context.Context, org string) (map[string]bool, error) {
	modules, err := a.repo.ActiveModules(ctx, org)
	if err != nil {
		return nil, err
	}
	for key, active := range modules {
		modules[key] = active && domain.ModuleCanActivate(key)
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
