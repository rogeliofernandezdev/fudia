package application

import (
	"context"
	"errors"
	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	identitycontracts "github.com/foods-platform/foods-backend/internal/modules/identity/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"regexp"
	"strings"
)

var ErrNotFound = errors.New("platform record not found")
var ErrConflict = errors.New("platform record conflict")

type scope = requestscope.Scope
type Filter struct {
	Query, PlanID, Standing string
	Page, Size              int
}
type OrganizationPage struct {
	Items           []domain.PlatformOrganizationView `json:"items"`
	Total           int                               `json:"total"`
	Page            int                               `json:"page"`
	PageSize        int                               `json:"pageSize"`
	Summary         map[string]int                    `json:"summary"`
	DueSoonDays     int                               `json:"dueSoonDays"`
	PlanOptions     []domain.OptionView               `json:"planOptions"`
	StandingOptions []domain.OptionView               `json:"standingOptions"`
}
type Platform struct {
	tips          billingcontracts.InitialTipConfigurator
	catalogs      FiscalCatalog
	subscriptions SubscriptionCreator
	passwords     PasswordHasher
	failures      FailureClassifier
	flow          FlowObserver
	repo          Repository
	work          unitofwork.Manager
}

func (a *Platform) WithTips(tips billingcontracts.InitialTipConfigurator) *Platform {
	a.tips = tips
	return a
}

func New(repo Repository, work unitofwork.Manager, catalogs FiscalCatalog, subscriptions SubscriptionCreator, passwords PasswordHasher, failures FailureClassifier, flow FlowObserver) *Platform {
	return &Platform{repo: repo, work: work, catalogs: catalogs, subscriptions: subscriptions, passwords: passwords, failures: failures, flow: flow}
}
func platformFailure(k apperror.Kind, c, m string) error { return apperror.New(k, c, m) }

type Repository interface {
	// CreateTenant crea empresa, perfil fiscal, primer local y administrador en un
	// solo envío; si falla devuelve *TenantCreateError con el paso afectado.
	CreateTenant(ctx context.Context, s scope, in domain.TenantOnboardingInput, currency orgcontracts.CurrencyOption, hash string) (domain.TenantRecords, error)
	AssignAdministrator(ctx context.Context, s scope, userID string, roleID string, locationID string) (int64, error)
	AuditOnboarded(ctx context.Context, s scope, organizationID string, locationID string, userID string) (int64, error)
	// SeedRoles inserta todos los roles en una sentencia y devuelve sus ids en el
	// orden recibido, incluidos los que ya existían.
	SeedRoles(context.Context, string, []identitycontracts.OrganizationRoleSeed) ([]string, error)
	SeedPaymentMethods(ctx context.Context, organizationID string) error
	SeedOperationalDefaults(ctx context.Context, organizationID string, locationID string, userID string, defaults domain.TenantOperationalDefaults) error
	MarkPastDue(ctx context.Context, s scope) (int64, error)
	Organizations(context.Context, Filter) (OrganizationPage, error)
	Organization(context.Context, string) (domain.PlatformOrganizationDetail, error)
	Channels(context.Context) ([]domain.PlatformWhatsAppChannel, error)
	CreateChannel(ctx context.Context, s scope, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error)
	UpdateChannel(ctx context.Context, s scope, id string, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error)
	CountryExists(context.Context, string) (bool, error)
	PlatformAdmin(context.Context, string) (bool, error)
}

func (a *Platform) Organizations(ctx context.Context, s scope, f Filter) (OrganizationPage, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.PlanID = strings.TrimSpace(f.PlanID)
	f.Standing = strings.TrimSpace(f.Standing)
	if f.Standing != "" && !domain.ValidPaymentStanding(f.Standing) {
		return OrganizationPage{}, platformFailure(apperror.Invalid, "invalid_standing", "La situación de pago no es válida.")
	}
	_, _ = a.repo.MarkPastDue(ctx, s)
	out, err := a.repo.Organizations(ctx, f)
	if err != nil {
		return out, platformFailure(apperror.Unavailable, "organizations_unavailable", "No pudimos cargar las empresas.")
	}
	out.DueSoonDays = domain.SubscriptionDueSoonDays
	out.StandingOptions = domain.PaymentStandingOptions
	return out, nil
}

var organizationIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (a *Platform) Organization(ctx context.Context, id string) (domain.PlatformOrganizationDetail, error) {
	var out domain.PlatformOrganizationDetail
	id = strings.TrimSpace(id)
	if !organizationIDPattern.MatchString(id) {
		return out, platformFailure(apperror.NotFound, "organization_not_found", "La empresa no existe.")
	}
	out, err := a.repo.Organization(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return out, platformFailure(apperror.NotFound, "organization_not_found", "La empresa no existe.")
	}
	if err != nil {
		return out, platformFailure(apperror.Unavailable, "organization_unavailable", "No pudimos cargar la empresa.")
	}
	return out, nil
}
func (a *Platform) Channels(ctx context.Context) ([]domain.PlatformWhatsAppChannel, error) {
	out, err := a.repo.Channels(ctx)
	if err != nil {
		return nil, platformFailure(apperror.Unavailable, "channels_unavailable", "No pudimos cargar los canales de WhatsApp.")
	}
	return out, nil
}
func (a *Platform) CreateChannel(ctx context.Context, s scope, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error) {
	var result domain.PlatformWhatsAppChannel
	if code, msg := domain.ValidatePlatformWhatsAppChannel(&in); code != "" {
		return result, platformFailure(apperror.Invalid, code, msg)
	}
	countryExists, err := a.repo.CountryExists(ctx, in.CountryCode)
	if err != nil {
		return result, platformFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos validar el país seleccionado.")
	}
	if !countryExists {
		return result, platformFailure(apperror.Invalid, "invalid_country", "El país no existe o está deshabilitado.")
	}
	var item domain.PlatformWhatsAppChannel
	err = func() error { var err error; item, err = a.repo.CreateChannel(ctx, s, in); return err }()
	if err != nil {
		return result, channelError(err)
	}
	return item, nil

}
func (a *Platform) UpdateChannel(ctx context.Context, s scope, id string, in domain.PlatformWhatsAppChannelInput) (domain.PlatformWhatsAppChannel, error) {
	var result domain.PlatformWhatsAppChannel
	if code, msg := domain.ValidatePlatformWhatsAppChannel(&in); code != "" {
		return result, platformFailure(apperror.Invalid, code, msg)
	}
	countryExists, err := a.repo.CountryExists(ctx, in.CountryCode)
	if err != nil {
		return result, platformFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos validar el país seleccionado.")
	}
	if !countryExists {
		return result, platformFailure(apperror.Invalid, "invalid_country", "El país no existe o está deshabilitado.")
	}
	var item domain.PlatformWhatsAppChannel
	err = func() error { var err error; item, err = a.repo.UpdateChannel(ctx, s, id, in); return err }()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return result, platformFailure(apperror.NotFound, "channel_not_found", "El canal de WhatsApp no existe.")
		}
		return result, channelError(err)
	}
	return item, nil

}
func (a *Platform) Authorize(ctx context.Context, id string) error {
	allowed, err := a.repo.PlatformAdmin(ctx, id)
	if err != nil || !allowed {
		return platformFailure(apperror.Forbidden, "platform_forbidden", "Solo el administrador de plataforma puede realizar esta acción.")
	}
	return nil
}
func channelError(err error) error {
	if errors.Is(err, ErrConflict) {
		return platformFailure(apperror.Conflict, "channel_already_exists", "El número ya está registrado en otro canal.")
	}
	return platformFailure(apperror.Internal, "channel_save_failed", "No pudimos guardar el canal de WhatsApp.")
}
