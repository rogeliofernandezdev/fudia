package contracts

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
)

type WhatsAppRoute = domain.WhatsAppRoute
type WhatsAppRoutes interface {
	WhatsAppRoutes(context.Context) ([]WhatsAppRoute, error)
}
