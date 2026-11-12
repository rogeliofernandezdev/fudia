package application

import (
	"context"
	"github.com/foods-platform/foods-backend/internal/modules/concierge/domain"
	platformcontracts "github.com/foods-platform/foods-backend/internal/modules/platformadmin/contracts"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"strings"
)

type RestaurantRepository interface {
	RestaurantDirectory(context.Context, string, string, domain.CustomerIdentity, bool, int) (domain.RestaurantPage, error)
	RestaurantContext(context.Context, string, string) (domain.RestaurantView, error)
}
type OriginRecorder interface {
	RecordConversationOrigin(context.Context, QRContext, domain.ConversationInput) error
}

func (a *Concierge) WithDirectory(repo RestaurantRepository, channels platformcontracts.WhatsAppRoutes, work unitofwork.Manager) *Concierge {
	a.directory = repo
	a.channels = channels
	a.work = work
	return a
}

func currentChannel(ctx context.Context, channels platformcontracts.WhatsAppRoutes, key string) (platformcontracts.WhatsAppRoute, error) {
	var out platformcontracts.WhatsAppRoute
	if channels == nil || strings.TrimSpace(key) == "" {
		return out, conciergeFailure(apperror.Unavailable, "channel_unavailable", "El canal no está disponible.")
	}
	routes, err := channels.WhatsAppRoutes(ctx)
	if err != nil {
		return out, conciergeFailure(apperror.Unavailable, "channel_unavailable", "No pudimos verificar el canal.")
	}
	for _, route := range routes {
		if route.RoutingKey == key && route.Active {
			return route, nil
		}
	}
	return out, conciergeFailure(apperror.Conflict, "channel_inactive", "Este canal ya no inicia atenciones nuevas. Escanea el QR vigente.")
}

func (a *Concierge) Restaurants(ctx context.Context, in domain.RestaurantSearchInput) (domain.RestaurantPage, error) {
	var out domain.RestaurantPage
	if !in.Identity.Valid() || in.Offset < 0 || in.Offset > 10000 {
		return out, conciergeFailure(apperror.Invalid, "invalid_customer_identity", "No pudimos validar la identidad del cliente.")
	}
	route, err := currentChannel(ctx, a.channels, in.ChannelKey)
	if err != nil {
		return out, err
	}
	if in.Identity.Provider != route.Provider {
		return out, conciergeFailure(apperror.Invalid, "identity_provider_mismatch", "La identidad no corresponde al canal.")
	}
	if a.directory == nil || a.work == nil {
		return out, conciergeFailure(apperror.Unavailable, "directory_unavailable", "El directorio no está disponible.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out, err = a.directory.RestaurantDirectory(tx.Context(), route.CountryCode, route.ProviderKey, in.Identity, in.OnlyVisited, in.Offset)
	if err != nil {
		return out, conciergeFailure(apperror.Unavailable, "directory_unavailable", "No pudimos consultar los restaurantes.")
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}

func (a *Concierge) RestaurantContext(ctx context.Context, in domain.RestaurantSelectionInput) (domain.RestaurantView, error) {
	var out domain.RestaurantView
	if !uuidPattern.MatchString(in.LocationID) {
		return out, conciergeFailure(apperror.Invalid, "invalid_restaurant", "Selecciona un restaurante de la lista.")
	}
	route, err := currentChannel(ctx, a.channels, in.ChannelKey)
	if err != nil {
		return out, err
	}
	if a.directory == nil {
		return out, conciergeFailure(apperror.Unavailable, "directory_unavailable", "El directorio no está disponible.")
	}
	out, err = a.directory.RestaurantContext(ctx, route.CountryCode, in.LocationID)
	if err != nil {
		return out, conciergeFailure(apperror.NotFound, "restaurant_unavailable", "El restaurante ya no está disponible. Elige otro de la lista.")
	}
	return out, nil
}

func (u *Usage) WithOrigins(origins OriginRecorder, channels platformcontracts.WhatsAppRoutes) *Usage {
	u.origins = origins
	u.channels = channels
	return u
}
