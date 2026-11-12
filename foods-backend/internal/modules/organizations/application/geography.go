package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"strings"
)

func (a *Organizations) GeographicAreas(ctx context.Context, parent string) ([]domain.GeographicArea, error) {
	parent = strings.TrimSpace(parent)
	if parent != "" {
		if len(parent) != 2 && len(parent) != 4 {
			return nil, organizationFailure(apperror.Invalid, "invalid_geographic_parent", "Selecciona un departamento o una provincia.")
		}
		for _, r := range parent {
			if r < '0' || r > '9' {
				return nil, organizationFailure(apperror.Invalid, "invalid_geographic_parent", "Selecciona una zona válida.")
			}
		}
	}
	items, err := a.repo.GeographicAreas(ctx, parent)
	if err != nil {
		return nil, organizationFailure(apperror.Unavailable, "geography_unavailable", "No pudimos cargar las zonas. Intenta nuevamente.")
	}
	return items, nil
}
