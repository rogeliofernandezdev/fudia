package application

import (
	"context"
	"errors"
	"strings"

	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type WhatsAppProvider interface {
	Verify(context.Context, domain.PlatformWhatsAppChannelInput) (domain.WhatsAppConnection, error)
	Link(context.Context, domain.WhatsAppLinkInput) (domain.WhatsAppConnection, error)
}

func (a *Platform) WithWhatsApp(provider WhatsAppProvider) *Platform { a.whatsapp = provider; return a }

func (a *Platform) saveWhatsAppChannel(ctx context.Context, s scope, id string, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error) {
	var out domain.PlatformWhatsAppChannel
	if code, msg := domain.ValidatePlatformWhatsAppChannel(&in); code != "" {
		return out, platformFailure(apperror.Invalid, code, msg)
	}
	exists, err := a.repo.CountryExists(ctx, in.CountryCode)
	if err != nil {
		return out, platformFailure(apperror.Unavailable, "country_unavailable", "No pudimos validar el país.")
	}
	if !exists {
		return out, platformFailure(apperror.Invalid, "invalid_country", "El país no existe o está deshabilitado.")
	}
	if id != "" {
		old, readErr := a.repo.Channel(ctx, id)
		if errors.Is(readErr, ErrNotFound) {
			return out, platformFailure(apperror.NotFound, "channel_not_found", "La configuración no existe.")
		}
		if readErr != nil {
			return out, channelError(readErr)
		}
		if old.CountryCode != in.CountryCode {
			return out, platformFailure(apperror.Invalid, "channel_country_immutable", "Edita la configuración del país correspondiente.")
		}
		if in.Revision != nil && *in.Revision != old.Revision {
			return out, platformFailure(apperror.Conflict, "channel_changed", "La configuración cambió. Recarga antes de guardar.")
		}
		in.Revision = &old.Revision
		if old.Provider == in.Provider && domain.WhatsAppDigits(old.PhoneNumber) == domain.WhatsAppDigits(in.PhoneNumber) {
			in.ProviderKey = old.ProviderKey
		}
	}
	if in.Active || (in.Provider == domain.Whatsmeow && in.SessionRef != "") {
		if a.whatsapp == nil {
			return out, platformFailure(apperror.Unavailable, "provider_unconfigured", "El servicio de WhatsApp no está configurado.")
		}
		if in.Provider == domain.Whatsmeow && in.SessionRef == "" {
			in.SessionRef = in.ProviderKey
		}
		connection, verifyErr := a.whatsapp.Verify(ctx, in)
		if verifyErr != nil {
			return out, platformFailure(apperror.Unavailable, "provider_unavailable", "No pudimos verificar el proveedor de WhatsApp.")
		}
		if connection.State != "connected" || connection.ProviderKey == "" || connection.CountryCode != in.CountryCode || connection.Provider != in.Provider || domain.WhatsAppDigits(connection.PhoneNumber) != domain.WhatsAppDigits(in.PhoneNumber) {
			return out, platformFailure(apperror.Conflict, "provider_not_ready", "Verifica el número en Meta o completa su vinculación antes de activar el canal.")
		}
		in.ProviderKey = connection.ProviderKey
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return out, channelError(err)
	}
	defer tx.Rollback(ctx)
	if id == "" {
		out, err = a.repo.CreateChannel(tx.Context(), s, in)
	} else {
		out, err = a.repo.UpdateChannel(tx.Context(), s, id, in)
	}
	if errors.Is(err, ErrNotFound) && id != "" {
		return out, platformFailure(apperror.Conflict, "channel_changed", "La configuración cambió. Recarga antes de guardar.")
	}
	if err != nil {
		return out, channelError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return out, channelError(err)
	}
	out.ConnectionState = "unconfigured"
	if in.Active {
		out.ConnectionState = "connected"
	}
	return out, nil
}

func (a *Platform) LinkWhatsApp(ctx context.Context, in domain.WhatsAppLinkInput) (domain.WhatsAppConnection, error) {
	var out domain.WhatsAppConnection
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	if len(in.CountryCode) != 2 || len(in.SessionRef) > 80 {
		return out, platformFailure(apperror.Invalid, "invalid_link", "Revisa el país y la vinculación.")
	}
	exists, err := a.repo.CountryExists(ctx, in.CountryCode)
	if err != nil || !exists {
		return out, platformFailure(apperror.Invalid, "invalid_country", "Selecciona un país habilitado.")
	}
	if a.whatsapp == nil {
		return out, platformFailure(apperror.Unavailable, "provider_unconfigured", "El gateway no está configurado.")
	}
	out, err = a.whatsapp.Link(ctx, in)
	if err != nil {
		return out, platformFailure(apperror.Unavailable, "provider_unavailable", "No pudimos consultar la vinculación de WhatsApp.")
	}
	out.ProviderKey = "" // Identificadores técnicos solo viajan entre servicios.
	return out, nil
}

func (a *Platform) WhatsAppRoutes(ctx context.Context) ([]domain.WhatsAppRoute, error) {
	return a.repo.WhatsAppRoutes(ctx)
}
func (a *Platform) ChannelHistory(ctx context.Context, id string) ([]map[string]any, error) {
	return a.repo.ChannelHistory(ctx, id)
}

func (a *Platform) RegisterLegacyRoute(ctx context.Context, in domain.WhatsAppRoute) error {
	if in.Provider != domain.MetaCloudAPI || len(in.CountryCode) != 2 || len(in.ProviderKey) < 4 || len(in.ProviderKey) > 80 || len(domain.WhatsAppDigits(in.PhoneNumber)) < 8 {
		return platformFailure(apperror.Invalid, "invalid_route", "La referencia del canal no es válida.")
	}
	in.RoutingKey = in.ProviderKey
	in.Legacy = true
	in.PhoneNumber = "+" + domain.WhatsAppDigits(in.PhoneNumber)
	return a.repo.RegisterLegacyRoute(ctx, in)
}
