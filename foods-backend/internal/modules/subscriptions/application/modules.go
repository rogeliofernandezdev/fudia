package application

import (
	"context"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/subscriptions/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

type ModuleRepository interface {
	Configured(context.Context, string) (map[string]bool, error)
	SetActive(context.Context, string, string, bool) error
}
type Modules struct {
	repo   ModuleRepository
	access *Access
	audit  audit.Repository
}

func NewModules(repo ModuleRepository, access *Access, audit audit.Repository) *Modules {
	return &Modules{repo, access, audit}
}
func (a *Modules) ActiveModules(ctx context.Context, org string) (map[string]bool, error) {
	return a.access.ActiveModules(ctx, org)
}
func (a *Modules) List(ctx context.Context, org string) ([]domain.ModuleView, error) {
	configured, err := a.repo.Configured(ctx, org)
	if err != nil {
		return nil, apperror.New(apperror.Unavailable, "modules_unavailable", "No pudimos cargar los módulos.")
	}
	items := make([]domain.ModuleView, 0, len(domain.ModuleCatalog))
	for _, m := range domain.ModuleCatalog {
		items = append(items, domain.ModuleView{ModuleDef: m, Availability: domain.ModuleAvailability(m.Key), Active: configured[m.Key] && domain.ModuleCanActivate(m.Key)})
	}
	return items, nil
}
func (a *Modules) SetActive(ctx context.Context, s requestscope.Scope, key string, active bool) error {
	if key == "" {
		return apperror.New(apperror.Invalid, "invalid_request", "Revisa los datos enviados.")
	}
	valid := false
	for _, m := range domain.ModuleCatalog {
		if m.Key == key {
			valid = true
			break
		}
	}
	if !valid {
		return apperror.New(apperror.Invalid, "invalid_module", "El módulo no existe.")
	}
	if active && !domain.ModuleCanActivate(key) {
		return apperror.New(apperror.Conflict, "module_not_ready", "Este módulo todavía no está disponible para empresas.")
	}
	if err := a.repo.SetActive(ctx, s.OrganizationID, key, active); err != nil {
		return apperror.New(apperror.Unavailable, "module_unavailable", "No pudimos actualizar el módulo.")
	}
	_ = a.audit.Record(ctx, s, "module.toggled", "module", key)
	return nil
}
