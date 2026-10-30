package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
	"math/big"
	"strings"
	"time"
)

var ErrNotFound = errors.New("organization record not found")

type scope = requestscope.Scope
type ServiceModeInput struct {
	ServiceMode string `json:"serviceMode"`
}
type OrganizationAccess interface {
	EnsureCapacity(context.Context, string, string) error
	ActiveModules(context.Context, string) (map[string]bool, error)
}
type Organizations struct {
	repo   OrganizationRepository
	work   unitofwork.Manager
	access OrganizationAccess
	audit  audit.Repository
}

func New(repo OrganizationRepository, work unitofwork.Manager, access OrganizationAccess, audit audit.Repository) *Organizations {
	return &Organizations{repo, work, access, audit}
}
func organizationFailure(k apperror.Kind, c, m string) error { return apperror.New(k, c, m) }

type LocationsPage struct {
	Items    []domain.LocationView `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}
type RatesPage struct {
	Items           []domain.ExchangeRate   `json:"items"`
	Total           int                     `json:"total"`
	Page            int                     `json:"page"`
	PageSize        int                     `json:"pageSize"`
	CurrencyOptions []domain.CurrencyOption `json:"currencyOptions"`
}
type ProfilesPage struct {
	Items           []domain.FiscalProfile  `json:"items"`
	Total           int                     `json:"total"`
	Page            int                     `json:"page"`
	PageSize        int                     `json:"pageSize"`
	CurrencyOptions []domain.CurrencyOption `json:"currencyOptions"`
	CountryOptions  []domain.CountryOption  `json:"countryOptions"`
}
type OrganizationRepository interface {
	Currencies(ctx context.Context) ([]domain.CurrencyOption, error)
	Countries(ctx context.Context) ([]domain.CountryOption, error)
	GlobalCountries(ctx context.Context) ([]domain.PlatformCountryOption, error)
	OnboardingCountries(ctx context.Context) ([]domain.PlatformCountryOption, error)
	CurrencyByCode(ctx context.Context, code string) (domain.CurrencyOption, bool, error)
	CountryByCode(ctx context.Context, code string) (domain.CountryOption, bool, error)
	OnboardingCountryByCode(ctx context.Context, code string) (domain.CountryOption, bool, error)
	Organization(ctx context.Context, s scope) (domain.OrganizationView, error)
	UpdateOrganization(ctx context.Context, s scope, in domain.OrganizationInput) (domain.OrganizationView, error)
	CreateLocation(ctx context.Context, s scope, in domain.LocationInput) (domain.LocationView, error)
	UpdateLocation(ctx context.Context, s scope, id string, in domain.LocationInput, active bool) (int64, error)
	DeactivateLocation(ctx context.Context, s scope, id string) (int64, error)
	DeactivateProfile(ctx context.Context, s scope, id string) (int64, error)
	CreateRate(ctx context.Context, s scope, in domain.ExchangeRateInput, effective time.Time) (domain.ExchangeRate, error)
	DeleteRate(ctx context.Context, s scope, id string) (int64, error)
	Settings(ctx context.Context, s scope) (domain.OrgSettings, error)
	LockDefaultProfile(ctx context.Context, s scope) (string, error)
	ClearDefaultProfile(ctx context.Context, s scope) (int64, error)
	SaveSettingsProfile(ctx context.Context, s scope, in domain.OrgSettingsInput, currency domain.CurrencyOption) (string, error)
	MigrateLocationProfile(ctx context.Context, s scope, profileID string, previousID string) (int64, error)
	CompleteSetup(ctx context.Context, s scope) (time.Time, error)
	ClearProfileDefaults(ctx context.Context, s scope) (int64, error)
	CreateProfile(ctx context.Context, s scope, in domain.FiscalProfileInput, currency domain.CurrencyOption) (string, error)
	UpdateProfile(ctx context.Context, s scope, id string, in domain.FiscalProfileInput, currency domain.CurrencyOption) (int64, error)
	DefaultProfileCount(ctx context.Context, s scope) (int, error)
	Locations(ctx context.Context, s scope, page int, size int) (LocationsPage, error)
	Rates(ctx context.Context, s scope, page int, size int) (RatesPage, error)
	Profiles(ctx context.Context, s scope, page int, size int) (ProfilesPage, error)
	Setup(ctx context.Context, s scope) (domain.RestaurantSetupView, error)
	SetServiceMode(ctx context.Context, s scope, in ServiceModeInput) (int64, error)
}

func (a *Organizations) Currencies(ctx context.Context) ([]domain.CurrencyOption, error) {
	return a.repo.Currencies(ctx)
}
func (a *Organizations) Countries(ctx context.Context) ([]domain.CountryOption, error) {
	return a.repo.Countries(ctx)
}
func (a *Organizations) GlobalCountries(ctx context.Context) ([]domain.PlatformCountryOption, error) {
	return a.repo.GlobalCountries(ctx)
}
func (a *Organizations) OnboardingCountries(ctx context.Context) ([]domain.PlatformCountryOption, error) {
	return a.repo.OnboardingCountries(ctx)
}
func (a *Organizations) CurrencyByCode(ctx context.Context, code string) (domain.CurrencyOption, bool, error) {
	return a.repo.CurrencyByCode(ctx, code)
}
func (a *Organizations) CountryByCode(ctx context.Context, code string) (domain.CountryOption, bool, error) {
	return a.repo.CountryByCode(ctx, code)
}
func (a *Organizations) OnboardingCountryByCode(ctx context.Context, code string) (domain.CountryOption, bool, error) {
	return a.repo.OnboardingCountryByCode(ctx, code)
}
func (a *Organizations) ValidateFiscal(ctx context.Context, in *domain.FiscalProfileInput, requireOperationalWhatsApp bool) (domain.CurrencyOption, bool, error) {
	if !domain.NormalizePlatformFiscalInput(in) {
		return domain.CurrencyOption{}, false, nil
	}

	var country domain.CountryOption
	var countryOK bool
	var err error
	if requireOperationalWhatsApp {
		country, countryOK, err = a.repo.OnboardingCountryByCode(ctx, in.Country)
	} else {
		country, countryOK, err = a.repo.CountryByCode(ctx, in.Country)
	}
	if err != nil || !countryOK {
		return domain.CurrencyOption{}, false, err
	}
	if requireOperationalWhatsApp && in.Currency != country.DefaultCurrency {
		return domain.CurrencyOption{}, false, nil
	}

	currency, currencyOK, err := a.repo.CurrencyByCode(ctx, in.Currency)
	if err != nil {
		return domain.CurrencyOption{}, false, err
	}
	return currency, currencyOK, nil
}
func (a *Organizations) Get(ctx context.Context, s scope) (domain.OrganizationView, error) {
	var result domain.OrganizationView
	var out domain.OrganizationView
	err := func() error { var err error; out, err = a.repo.Organization(ctx, s); return err }()
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "organization_unavailable", "No pudimos cargar los datos de la empresa.")
	}
	return out, nil

}
func (a *Organizations) Update(ctx context.Context, s scope, in domain.OrganizationInput) (domain.OrganizationView, error) {
	var result domain.OrganizationView
	in.LegalName, in.TradeName, in.TaxID, in.Timezone = strings.TrimSpace(in.LegalName), strings.TrimSpace(in.TradeName), strings.TrimSpace(in.TaxID), strings.TrimSpace(in.Timezone)
	if in.LegalName == "" || in.TradeName == "" || len(in.TaxID) < 6 || len(in.TaxID) > 32 || in.Timezone == "" {
		return result, organizationFailure(apperror.Invalid, "invalid_organization", "Completa los datos obligatorios de la empresa.")
	}
	var out domain.OrganizationView
	err := func() error { var err error; out, err = a.repo.UpdateOrganization(ctx, s, in); return err }()
	if err != nil {
		return result, organizationFailure(apperror.Conflict, "organization_conflict", "No pudimos guardar la empresa. Verifica su identificación fiscal.")
	}
	_ = a.audit.Record(ctx, s, "organization.updated", "organization", out.ID)
	return out, nil

}
func (a *Organizations) CreateLocation(ctx context.Context, s scope, in domain.LocationInput) (domain.LocationView, error) {
	var result domain.LocationView
	if !domain.ValidateLocationInput(&in) {
		return result, organizationFailure(apperror.Invalid, "invalid_location", "Completa los datos obligatorios del local.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "location_unavailable", "No pudimos crear el local.")
	}
	defer tx.Rollback(ctx)
	if err = a.access.EnsureCapacity(tx.Context(), s.OrganizationID, "locations"); err != nil {
		return result, err
	}
	var out domain.LocationView
	err = func() error { var err error; out, err = a.repo.CreateLocation(tx.Context(), s, in); return err }()
	if err != nil {
		return result, organizationFailure(apperror.Conflict, "location_conflict", "El código ya existe o el perfil fiscal no pertenece a la empresa.")
	}
	if err = tx.Commit(ctx); err != nil {
		return result, organizationFailure(apperror.Unavailable, "location_unavailable", "No pudimos confirmar el nuevo local.")
	}
	_ = a.audit.Record(ctx, s, "location.created", "location", out.ID)
	return out, nil

}
func (a *Organizations) UpdateLocation(ctx context.Context, s scope, id string, in domain.LocationInput) error {
	if !domain.ValidateLocationInput(&in) {
		return organizationFailure(apperror.Invalid, "invalid_location", "Completa los datos obligatorios del local.")
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	command, err := a.repo.UpdateLocation(ctx, s, id, in, active)
	if err != nil || command == 0 {
		return organizationFailure(apperror.Conflict, "location_conflict", "No pudimos actualizar el local.")
	}
	_ = a.audit.Record(ctx, s, "location.updated", "location", id)
	return nil

}
func (a *Organizations) DeactivateLocation(ctx context.Context, s scope, id string) error {
	if id == s.LocationID {
		return organizationFailure(apperror.Conflict, "active_location", "No puedes desactivar el local de tu sesión actual.")
	}
	command, err := a.repo.DeactivateLocation(ctx, s, id)
	if err != nil || command == 0 {
		return organizationFailure(apperror.NotFound, "location_not_found", "No encontramos el local activo.")
	}
	_ = a.audit.Record(ctx, s, "location.deactivated", "location", id)
	return nil

}
func (a *Organizations) DeactivateProfile(ctx context.Context, s scope, id string) error {
	command, err := a.repo.DeactivateProfile(ctx, s, id)
	if err != nil || command == 0 {
		return organizationFailure(apperror.Conflict, "profile_in_use", "No puedes desactivar el perfil predeterminado ni uno asignado a locales activos.")
	}
	_ = a.audit.Record(ctx, s, "fiscal_profile.deactivated", "fiscal_profile", id)
	return nil

}
func (a *Organizations) CreateRate(ctx context.Context, s scope, in domain.ExchangeRateInput) (domain.ExchangeRate, error) {
	var result domain.ExchangeRate
	in.BaseCurrency, in.QuoteCurrency, in.Rate, in.Source = strings.ToUpper(strings.TrimSpace(in.BaseCurrency)), strings.ToUpper(strings.TrimSpace(in.QuoteCurrency)), strings.TrimSpace(in.Rate), strings.TrimSpace(in.Source)
	rate, rateOK := new(big.Rat).SetString(in.Rate)
	effective, timeErr := time.Parse(time.RFC3339, in.EffectiveAt)
	_, baseOK, baseErr := a.repo.CurrencyByCode(ctx, in.BaseCurrency)
	_, quoteOK, quoteErr := a.repo.CurrencyByCode(ctx, in.QuoteCurrency)
	if baseErr != nil || quoteErr != nil {
		return result, organizationFailure(apperror.Unavailable, "rates_unavailable", "No pudimos validar el catálogo de monedas.")
	}
	if !baseOK || !quoteOK || in.BaseCurrency == in.QuoteCurrency || !rateOK || rate.Sign() <= 0 || timeErr != nil || (in.Source != "manual" && in.Source != "provider") {
		return result, organizationFailure(apperror.Invalid, "invalid_rate", "Revisa las monedas, la tasa, la fecha y la fuente.")
	}
	var out domain.ExchangeRate
	err := func() error { var err error; out, err = a.repo.CreateRate(ctx, s, in, effective); return err }()
	if err != nil {
		return result, organizationFailure(apperror.Conflict, "rate_conflict", "Ya existe un tipo de cambio para ese par y fecha.")
	}
	out.CreatedByName = &s.Name
	_ = a.audit.Record(ctx, s, "exchange_rate.created", "exchange_rate", out.ID)
	return out, nil

}
func (a *Organizations) DeleteRate(ctx context.Context, s scope, id string) error {
	command, err := a.repo.DeleteRate(ctx, s, id)
	if err != nil || command == 0 {
		return organizationFailure(apperror.NotFound, "rate_not_found", "No encontramos el tipo de cambio.")
	}
	_ = a.audit.Record(ctx, s, "exchange_rate.deleted", "exchange_rate", id)
	return nil

}
func (a *Organizations) Settings(ctx context.Context, s scope) (domain.OrgSettings, error) {
	var result domain.OrgSettings
	var out domain.OrgSettings
	err := func() error { var err error; out, err = a.repo.Settings(ctx, s); return err }()
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "settings_unavailable", "No pudimos cargar la configuración financiera.")
	}
	out.CurrencyOptions, err = a.repo.Currencies(ctx)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "settings_unavailable", "No pudimos cargar las monedas configuradas en la plataforma.")
	}
	out.CountryOptions, err = a.repo.Countries(ctx)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "settings_unavailable", "No pudimos cargar los países configurados en la plataforma.")
	}
	return out, nil

}
func (a *Organizations) UpdateSettings(ctx context.Context, s scope, in domain.OrgSettingsInput) (domain.OrgSettings, error) {
	var result domain.OrgSettings
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Country = strings.ToUpper(strings.TrimSpace(in.Country))
	in.CurrencyPosition = strings.TrimSpace(in.CurrencyPosition)
	in.TaxName = strings.TrimSpace(in.TaxName)
	currency, validCurrency, currencyErr := a.repo.CurrencyByCode(ctx, in.Currency)
	_, validCountry, countryErr := a.repo.CountryByCode(ctx, in.Country)
	if currencyErr != nil || countryErr != nil {
		return result, organizationFailure(apperror.Unavailable, "settings_unavailable", "No pudimos validar los catálogos de país y moneda.")
	}
	if !validCountry || !validCurrency || (in.CurrencyPosition != "before" && in.CurrencyPosition != "after") || in.TaxName == "" || len(in.TaxName) > 30 || in.TaxRate < 0 || in.TaxRate > 1 {
		return result, organizationFailure(apperror.Invalid, "invalid_settings", "Revisa la moneda y el porcentaje de impuesto.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err == nil {
		defer tx.Rollback(ctx)
		var previousID string
		err = func() error { var err error; previousID, err = a.repo.LockDefaultProfile(tx.Context(), s); return err }()
		if err == nil {
			_, err = a.repo.ClearDefaultProfile(tx.Context(), s)
		}
		var profileID string
		if err == nil {
			err = func() error {
				var err error
				profileID, err = a.repo.SaveSettingsProfile(tx.Context(), s, in, currency)
				return err
			}()
		}
		if err == nil {
			_, err = a.repo.MigrateLocationProfile(tx.Context(), s, profileID, previousID)
		}
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
	}
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "settings_unavailable", "No pudimos guardar la configuración financiera.")
	}
	_ = a.audit.Record(ctx, s, "organization.settings.updated", "organization", s.OrganizationID)
	return a.Settings(ctx, s)

}
func (a *Organizations) CompleteSetup(ctx context.Context, s scope) (domain.RestaurantSetupView, error) {
	var result domain.RestaurantSetupView
	out, err := a.setup(ctx, s)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "restaurant_setup_unavailable", "No pudimos validar la puesta en marcha del restaurante.")
	}
	if !out.CoreReady {
		return result, organizationFailure(apperror.Conflict, "restaurant_setup_incomplete", "Completa el tipo de atención, la carta y las mesas requeridas antes de finalizar.")
	}
	var completedAt time.Time
	err = func() error { var err error; completedAt, err = a.repo.CompleteSetup(ctx, s); return err }()
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "restaurant_setup_unavailable", "No pudimos finalizar la puesta en marcha.")
	}
	_ = a.audit.Record(ctx, s, "restaurant_setup.completed", "organization", s.OrganizationID)
	out.CompletedAt = &completedAt
	return out, nil

}
func (a *Organizations) SaveProfile(ctx context.Context, s scope, id string, in domain.FiscalProfileInput, create bool) (map[string]string, error) {
	var result map[string]string
	currency, valid, catalogErr := a.ValidateFiscal(ctx, &in, false)
	if catalogErr != nil {
		return result, organizationFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos validar los catálogos de país y moneda.")
	}
	if !valid {
		return result, organizationFailure(apperror.Invalid, "invalid_profile", "Revisa país, moneda e impuesto.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "profile_unavailable", "No pudimos guardar el perfil fiscal.")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if in.Default {
		_, err = a.repo.ClearProfileDefaults(tx.Context(), s)
	}
	var savedID string
	if create && err == nil {
		err = func() error {
			var err error
			savedID, err = a.repo.CreateProfile(tx.Context(), s, in, currency)
			return err
		}()
	} else if err == nil {
		savedID = id
		command, updateErr := a.repo.UpdateProfile(tx.Context(), s, savedID, in, currency)
		err = updateErr
		if err == nil && command == 0 {
			err = ErrNotFound
		}
	}
	if err == nil && !in.Default {
		var defaults int
		err = func() error { var err error; defaults, err = a.repo.DefaultProfileCount(tx.Context(), s); return err }()
		if err == nil && defaults == 0 {
			err = errors.New("default fiscal profile required")
		}
	}
	if err != nil || tx.Commit(ctx) != nil {
		return result, organizationFailure(apperror.Conflict, "profile_conflict", "El país ya existe o la empresa debe conservar un perfil predeterminado.")
	}
	_ = a.audit.Record(ctx, s, map[bool]string{true: "fiscal_profile.created", false: "fiscal_profile.updated"}[create], "fiscal_profile", savedID)
	return map[string]string{"id": savedID}, nil
}
func (a *Organizations) Locations(ctx context.Context, s scope, page int, size int) (LocationsPage, error) {
	var result LocationsPage
	out, err := a.repo.Locations(ctx, s, page, size)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "locations_unavailable", "No pudimos cargar los locales.")
	}
	return out, nil
}
func (a *Organizations) Rates(ctx context.Context, s scope, page int, size int) (RatesPage, error) {
	var result RatesPage
	currencies, err := a.repo.Currencies(ctx)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "rates_unavailable", "No pudimos cargar el catálogo de monedas.")
	}
	out, err := a.repo.Rates(ctx, s, page, size)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "rates_unavailable", "No pudimos cargar los tipos de cambio.")
	}
	out.CurrencyOptions = currencies
	return out, nil
}
func (a *Organizations) Profiles(ctx context.Context, s scope, page int, size int) (ProfilesPage, error) {
	var result ProfilesPage
	countries, err := a.repo.Countries(ctx)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "profiles_unavailable", "No pudimos cargar el catálogo de países.")
	}
	currencies, err := a.repo.Currencies(ctx)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "profiles_unavailable", "No pudimos cargar el catálogo de monedas.")
	}
	countryNames := make(map[string]string, len(countries))
	for _, country := range countries {
		countryNames[country.Code] = country.Name
	}
	out, err := a.repo.Profiles(ctx, s, page, size)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "profiles_unavailable", "No pudimos cargar los perfiles fiscales.")
	}
	out.CurrencyOptions = currencies
	out.CountryOptions = countries
	for i := range out.Items {
		out.Items[i].CountryName = out.Items[i].Country
		if name, ok := countryNames[out.Items[i].Country]; ok {
			out.Items[i].CountryName = name
		}
	}
	return out, nil
}
func (a *Organizations) setup(ctx context.Context, s scope) (domain.RestaurantSetupView, error) {
	out, err := a.repo.Setup(ctx, s)
	if err != nil {
		return out, err
	}
	out.Modules, err = a.access.ActiveModules(ctx, s.OrganizationID)
	if err != nil {
		return out, err
	}
	out.CoreReady = domain.RestaurantSetupCoreReady(out.ServiceMode, out.Counts)
	return out, nil
}
func (a *Organizations) Setup(ctx context.Context, s scope) (domain.RestaurantSetupView, error) {
	var result domain.RestaurantSetupView
	out, err := a.setup(ctx, s)
	if errors.Is(err, ErrNotFound) {
		return result, organizationFailure(apperror.NotFound, "restaurant_setup_not_found", "La empresa todavía no tiene una puesta en marcha registrada.")
	}
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "restaurant_setup_unavailable", "No pudimos cargar la puesta en marcha del restaurante.")
	}
	return out, nil

}
func (a *Organizations) SetServiceMode(ctx context.Context, s scope, in ServiceModeInput) (domain.RestaurantSetupView, error) {
	var result domain.RestaurantSetupView
	in.ServiceMode = strings.TrimSpace(in.ServiceMode)
	if !domain.ValidServiceMode(in.ServiceMode) {
		return result, organizationFailure(apperror.Invalid, "invalid_service_mode", "Selecciona atención en mostrador, salón o ambas.")
	}
	tag, err := a.repo.SetServiceMode(ctx, s, in)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "restaurant_setup_unavailable", "No pudimos guardar el tipo de atención.")
	}
	if tag == 0 {
		return result, organizationFailure(apperror.NotFound, "restaurant_setup_not_found", "La empresa todavía no tiene una puesta en marcha registrada.")
	}
	_ = a.audit.Record(ctx, s, "restaurant_setup.service_mode_updated", "organization", s.OrganizationID)
	return a.Setup(ctx, s)

}
func (a *Organizations) OnboardingCatalogs(ctx context.Context, all bool) (map[string]any, error) {
	var result map[string]any
	var countries any
	if all {
		items, err := a.repo.GlobalCountries(ctx)
		if err != nil {
			return result, organizationFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos cargar los países disponibles.")
		}
		countries = items
	} else {
		items, err := a.repo.OnboardingCountries(ctx)
		if err != nil {
			return result, organizationFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos cargar los países disponibles.")
		}
		countries = items
	}
	currencies, err := a.repo.Currencies(ctx)
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "catalogs_unavailable", "No pudimos cargar las monedas disponibles.")
	}
	return map[string]any{"countryOptions": countries, "currencyOptions": currencies, "tipOptions": billingcontracts.OnboardingTipOptions(), "maxMenuPages": 20}, nil

}
