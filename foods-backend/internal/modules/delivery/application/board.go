package application

import (
	"context"
	"errors"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// Opción «Delivery» del restaurante: lecturas con el alcance de empresa y local
// de la sesión. Nunca exponen datos de otro local ni permiten editar centrales.

var errBoardScope = failure(apperror.Forbidden, "location_required", "Selecciona un local para ver su delivery.")

func boardUnavailable(err error) error {
	return unavailable("delivery_board_unavailable", "No pudimos cargar el delivery del local.", err)
}

// Board lista los deliveries del local con su despacho a las centrales.
func (a *Dispatcher) Board(ctx context.Context, s scope) ([]domain.BoardOrder, error) {
	if s.OrganizationID == "" || s.LocationID == "" {
		return nil, errBoardScope
	}
	out, err := a.panel.BoardOrders(ctx, s)
	if err != nil {
		return nil, boardUnavailable(err)
	}
	return out, nil
}

// Partners devuelve las centrales activas que cubren el distrito del local.
func (a *Dispatcher) Partners(ctx context.Context, s scope) (domain.LocationPartners, error) {
	if s.OrganizationID == "" || s.LocationID == "" {
		return domain.LocationPartners{}, errBoardScope
	}
	out, err := a.panel.LocationPartners(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return out, failure(apperror.NotFound, "location_not_found", "El local no existe.")
	}
	if err != nil {
		return out, boardUnavailable(err)
	}
	return out, nil
}

// FailedDeliveries lista las entregas fallidas del local.
func (a *Dispatcher) FailedDeliveries(ctx context.Context, s scope) ([]domain.LocationFailedDelivery, error) {
	if s.OrganizationID == "" || s.LocationID == "" {
		return nil, errBoardScope
	}
	out, err := a.panel.LocationFailedDeliveries(ctx, s)
	if err != nil {
		return nil, boardUnavailable(err)
	}
	return out, nil
}
