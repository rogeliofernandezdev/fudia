package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/modules/delivery/domain"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// Tracking es el seguimiento de un pedido de delivery para el cliente: origen,
// destino y, mientras una central comparte su ubicación, dónde va el repartidor.
type Tracking = domain.Tracking

// TrackingPoint es una coordenada del seguimiento.
type TrackingPoint = domain.Point

// TrackingReader publica el seguimiento a los canales del cliente (Concierge).
// found=false si el pedido no existe en ese local o no es de delivery.
type TrackingReader interface {
	DeliveryTracking(ctx context.Context, s requestscope.Scope, orderID string) (Tracking, bool, error)
}
