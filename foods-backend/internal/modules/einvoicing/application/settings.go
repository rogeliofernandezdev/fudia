package application

import (
	"context"
	"errors"
	"strings"

	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/domain"
	"github.com/foods-platform/foods-backend/internal/modules/einvoicing/ports"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

type ProfileSettingsInput struct {
	Provider    string `json:"provider"`
	Environment string `json:"environment"`
}
type LocationSettingsInput struct {
	DefaultDocument string            `json:"defaultDocument"`
	Series          map[string]string `json:"series"`
}

type ProfileSettings struct {
	FiscalProfileID string `json:"fiscalProfileId"`
	Country         string `json:"country"`
	CountryName     string `json:"countryName"`
	Currency        string `json:"currency"`
	Default         bool   `json:"default"`
	Supported       bool   `json:"supported"`
	Provider        string `json:"provider"`
	Environment     string `json:"environment"`
	// Credentials of the selected provider; IssuerIssue blocks issuing.
	Credentials CredentialStatus `json:"credentials"`
	IssuerIssue string           `json:"issuerIssue"`
	// Ready: with the module active, the profile issues as soon as issuer and
	// credentials are complete (each location still needs its series). Issue
	// explains what is missing.
	Ready bool   `json:"ready"`
	Issue string `json:"issue"`
}
type LocationSeries struct {
	DocumentType string `json:"documentType"`
	Label        string `json:"label"`
	Series       string `json:"series"`
	Suggested    string `json:"suggested"`
	LastNumber   int    `json:"lastNumber"`
}
type LocationSettings struct {
	LocationID      string           `json:"locationId"`
	Name            string           `json:"name"`
	FiscalProfileID string           `json:"fiscalProfileId"`
	Country         string           `json:"country"`
	Supported       bool             `json:"supported"`
	DefaultDocument string           `json:"defaultDocument"`
	Series          []LocationSeries `json:"series"`
}
type SettingsView struct {
	Entitled         bool               `json:"entitled"`
	Profiles         []ProfileSettings  `json:"profiles"`
	Locations        []LocationSettings `json:"locations"`
	ProviderOptions  []ports.Option     `json:"providerOptions"`
	EnvironmentLabel map[string]string  `json:"environmentLabels"`
}

type Settings struct {
	repo     Repository
	work     unitofwork.Manager
	audit    audit.Repository
	access   Entitlement
	registry *ports.Registry
	cipher   Cipher
}

func NewSettings(repo Repository, work unitofwork.Manager, audit audit.Repository, access Entitlement, registry *ports.Registry) *Settings {
	return &Settings{repo: repo, work: work, audit: audit, access: access, registry: registry}
}
func (a *Settings) WithCipher(c Cipher) *Settings { a.cipher = c; return a }

func (a *Settings) View(ctx context.Context, s scope) (SettingsView, error) {
	out := SettingsView{Profiles: []ProfileSettings{}, Locations: []LocationSettings{}, ProviderOptions: a.registry.ProviderOptions(),
		EnvironmentLabel: map[string]string{domain.EnvironmentTest: "Pruebas", domain.EnvironmentProduction: "Producción"}}
	entitled, err := a.access.ModuleEntitled(ctx, s.OrganizationID, domain.ModuleKey)
	if err != nil {
		return out, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar la facturación electrónica.")
	}
	out.Entitled = entitled
	profiles, locations, err := a.repo.Settings(ctx, s)
	if err != nil {
		return out, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos cargar la configuración de facturación.")
	}
	defaultProvider := ""
	if options := a.registry.ProviderOptions(); len(options) > 0 {
		defaultProvider = options[0].Value
	}
	issuer, err := a.repo.Issuer(ctx, s)
	if err != nil {
		return out, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos cargar los datos del emisor.")
	}
	for _, p := range profiles {
		j, supported := a.registry.Jurisdiction(p.Country)
		if p.Provider == "" {
			p.Provider, p.Environment = defaultProvider, domain.EnvironmentTest
		}
		item := ProfileSettings{FiscalProfileID: p.FiscalProfileID, Country: p.Country, CountryName: p.CountryName, Currency: p.Currency, Default: p.Default, Supported: supported, Provider: p.Provider, Environment: p.Environment}
		if supported {
			if issue := j.ValidateIssuer(issuer); issue != nil {
				item.IssuerIssue = issue.Message
				item.Issue = issue.Message
			}
		}
		if provider, ok := a.registry.Provider(p.Provider); ok {
			if item.Credentials, err = a.credentialStatus(ctx, s, p.FiscalProfileID, provider, p.Environment); err != nil {
				return out, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos cargar las credenciales.")
			}
			if supported && item.Issue == "" {
				var known *apperror.Error
				if err := verifyCredentials(ctx, a.repo, a.cipher, s, p.FiscalProfileID, provider, p.Environment); errors.As(err, &known) && known.Kind == apperror.Conflict {
					item.Issue = known.Message
				} else if err != nil {
					return out, err
				}
			}
		} else if supported && item.Issue == "" {
			item.Issue = "Elige el proveedor de facturación."
		}
		item.Ready = entitled && supported && item.Issue == ""
		out.Profiles = append(out.Profiles, item)
	}
	for _, l := range locations {
		item := LocationSettings{LocationID: l.LocationID, Name: l.Name, FiscalProfileID: l.FiscalProfileID, Country: l.Country, DefaultDocument: l.DefaultDocument, Series: []LocationSeries{}}
		if j, ok := a.registry.Jurisdiction(l.Country); ok {
			item.Supported = true
			for _, option := range j.DocumentTypes() {
				current := l.Series[option.Value]
				item.Series = append(item.Series, LocationSeries{DocumentType: option.Value, Label: option.Label, Series: current.Series, Suggested: j.DefaultSeries(option.Value, l.Position), LastNumber: current.LastNumber})
			}
		}
		out.Locations = append(out.Locations, item)
	}
	return out, nil
}

func (a *Settings) SaveProfile(ctx context.Context, s scope, profileID string, in ProfileSettingsInput) (SettingsView, error) {
	in.Provider, in.Environment = strings.TrimSpace(in.Provider), strings.TrimSpace(in.Environment)
	provider, ok := a.registry.Provider(in.Provider)
	if !ok || (in.Environment != domain.EnvironmentTest && in.Environment != domain.EnvironmentProduction) {
		return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_settings", "Elige el proveedor y el ambiente.")
	}
	if provider.TestOnly() && in.Environment == domain.EnvironmentProduction {
		return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_settings", "El proveedor simulado solo funciona en el ambiente de pruebas.")
	}
	country, err := a.repo.ProfileCountry(ctx, s, profileID)
	if errors.Is(err, ErrNotFound) {
		return SettingsView{}, failure(apperror.NotFound, "profile_not_found", "No encontramos el perfil fiscal.")
	}
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar el perfil fiscal.")
	}
	if _, supported := a.registry.Jurisdiction(country); !supported {
		return SettingsView{}, failure(apperror.Conflict, "einvoicing_unsupported_country", "La facturación electrónica aún no está disponible para este país.")
	}
	entitled, err := a.access.ModuleEntitled(ctx, s.OrganizationID, domain.ModuleKey)
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar la facturación electrónica.")
	}
	if !entitled {
		return SettingsView{}, failure(apperror.Conflict, "einvoicing_not_entitled", "Tu plan no incluye facturación electrónica.")
	}
	if err := a.repo.SaveProfileSettings(ctx, s, profileID, in); err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos guardar la configuración de facturación.")
	}
	_ = a.audit.Record(ctx, s, "einvoicing.profile.updated", "fiscal_profile", profileID)
	return a.View(ctx, s)
}

func (a *Settings) SaveLocation(ctx context.Context, s scope, locationID string, in LocationSettingsInput) (SettingsView, error) {
	in.DefaultDocument = strings.TrimSpace(in.DefaultDocument)
	if in.DefaultDocument == "" {
		in.DefaultDocument = domain.TypeNone
	}
	country, err := a.repo.LocationCountry(ctx, s, locationID)
	if errors.Is(err, ErrNotFound) {
		return SettingsView{}, failure(apperror.NotFound, "location_not_found", "No encontramos el local.")
	}
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar el local.")
	}
	j, ok := a.registry.Jurisdiction(country)
	if !ok {
		return SettingsView{}, failure(apperror.Conflict, "einvoicing_unsupported_country", "La facturación electrónica aún no está disponible para el país de este local.")
	}
	if in.DefaultDocument != domain.TypeNone && !offered(j, in.DefaultDocument) {
		return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_settings", "Elige el comprobante sugerido al cobrar.")
	}
	series := map[string]string{}
	for _, option := range j.DocumentTypes() {
		value := strings.ToUpper(strings.TrimSpace(in.Series[option.Value]))
		if !j.ValidSeries(option.Value, value) {
			return SettingsView{}, failure(apperror.Invalid, "invalid_einvoicing_series", "Revisa la serie de "+strings.ToLower(option.Label)+": "+seriesHint(j, option.Value)+".")
		}
		series[option.Value] = value
	}
	in.Series = series
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos guardar las series.")
	}
	defer tx.Rollback(ctx)
	err = a.repo.SaveLocationSettings(tx.Context(), s, locationID, in)
	switch {
	case errors.Is(err, ErrSeriesInUse):
		return SettingsView{}, failure(apperror.Conflict, "einvoicing_series_in_use", "Una serie con comprobantes emitidos no se puede cambiar.")
	case errors.Is(err, ErrSeriesTaken):
		return SettingsView{}, failure(apperror.Conflict, "einvoicing_series_taken", "Esa serie ya la usa otro local de la empresa.")
	case err != nil:
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos guardar las series.")
	}
	if err := tx.Commit(ctx); err != nil {
		return SettingsView{}, failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos confirmar las series.")
	}
	_ = a.audit.Record(ctx, s, "einvoicing.location.updated", "location", locationID)
	return a.View(ctx, s)
}

// verifyCredentials requires the credentials the provider needs in that
// environment and re-validates them (e.g. production needs a certificate that
// beta did not). A missing or invalid credential is a Conflict with a message
// for the user.
func verifyCredentials(ctx context.Context, repo Repository, cipher Cipher, s scope, profileID string, provider ports.Provider, environment string) error {
	sealed, found, err := repo.Credentials(ctx, s, profileID, provider.Key())
	if err != nil {
		return failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos leer las credenciales.")
	}
	if !found {
		if provider.RequiresCredentials(environment) {
			return failure(apperror.Conflict, "einvoicing_credentials_missing", "Configura las credenciales de "+provider.Label()+" para este ambiente.")
		}
		return nil
	}
	values, err := openCredentials(cipher, sealed, s.OrganizationID, profileID, provider.Key())
	if err != nil {
		return failure(apperror.Unavailable, "einvoicing_credentials_unavailable", "No pudimos descifrar las credenciales guardadas.")
	}
	_, err = provider.ValidateCredentials(environment, values)
	var credentialErr *ports.CredentialError
	if errors.As(err, &credentialErr) {
		return failure(apperror.Conflict, "einvoicing_credentials_missing", credentialErr.Message)
	}
	if err != nil {
		return failure(apperror.Unavailable, "einvoicing_unavailable", "No pudimos validar las credenciales.")
	}
	return nil
}

func seriesHint(j ports.Jurisdiction, documentType string) string {
	return "por ejemplo " + j.DefaultSeries(documentType, 1)
}
