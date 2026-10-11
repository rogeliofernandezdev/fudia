package application

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/inventory/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

var (
	ErrUnitConflict      = errors.New("unit conflict")
	ErrUnitPresentations = errors.New("unit presentations unavailable")
)

// UnitRepository creates the unit, presentation combinations and audit as one aggregate transaction.
type UnitRepository interface {
	List(context.Context, string) ([]domain.Unit, error)
	Create(context.Context, requestscope.Scope, domain.UnitInput) (domain.Unit, error)
}
type UnitService struct{ repository UnitRepository }

func NewUnits(repository UnitRepository) *UnitService { return &UnitService{repository} }
func (s *UnitService) List(ctx context.Context, organizationID string) ([]domain.Unit, error) {
	out, err := s.repository.List(ctx, organizationID)
	if err != nil {
		return nil, unavailable("No pudimos cargar las unidades.")
	}
	return out, nil
}
func (s *UnitService) Create(ctx context.Context, scope requestscope.Scope, in domain.UnitInput) (domain.Unit, error) {
	in, invalid := domain.NormalizeUnit(in)
	if invalid != "" {
		return domain.Unit{}, apperror.New(apperror.Invalid, "invalid_inventory_unit", invalid)
	}
	out, err := s.repository.Create(ctx, scope, in)
	switch {
	case errors.Is(err, ErrUnitConflict):
		return out, apperror.New(apperror.Conflict, "inventory_unit_conflict", "Ya existe una unidad con ese código.")
	case errors.Is(err, ErrUnitPresentations):
		return out, unavailable("No pudimos preparar las presentaciones de la unidad.")
	case err != nil:
		return out, unavailable("No pudimos guardar la unidad.")
	}
	return out, nil
}
func unavailable(message string) error {
	return apperror.New(apperror.Unavailable, "inventory_units_unavailable", message)
}
