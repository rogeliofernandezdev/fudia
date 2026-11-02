package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
)

// TaxProfile is the tax of a fiscal profile: its general rate and, when the
// company is enrolled, the special regime that replaces it while it has a rate.
type TaxProfile struct {
	General       domain.TaxComponent
	TaxIncluded   bool
	SpecialRegime *domain.SpecialTaxRegime
}
type TaxRepository interface {
	TaxProfile(ctx context.Context, s scope, profileID string) (TaxProfile, error)
	// BusinessDate is today in the session location's timezone.
	BusinessDate(ctx context.Context, s scope) (string, error)
}
type Taxes struct{ repo TaxRepository }

func NewTaxes(repo TaxRepository) *Taxes { return &Taxes{repo} }

// EffectiveTax resolves the tax that applies to a profile on a business date;
// an empty date means today in the session location's timezone.
func (a *Taxes) EffectiveTax(ctx context.Context, s scope, profileID, date string) (domain.EffectiveTax, error) {
	if date != "" && !domain.ValidBusinessDate(date) {
		return domain.EffectiveTax{}, organizationFailure(apperror.Invalid, "invalid_tax_date", "Revisa la fecha consultada.")
	}
	profile, err := a.repo.TaxProfile(ctx, s, profileID)
	if errors.Is(err, ErrNotFound) {
		return domain.EffectiveTax{}, organizationFailure(apperror.NotFound, "profile_not_found", "No encontramos el perfil fiscal.")
	}
	if err == nil && date == "" {
		date, err = a.repo.BusinessDate(ctx, s)
	}
	if err != nil {
		return domain.EffectiveTax{}, organizationFailure(apperror.Unavailable, "taxes_unavailable", "No pudimos calcular el impuesto vigente.")
	}
	return domain.ResolveEffectiveTax(date, profile.General, profile.TaxIncluded, profile.SpecialRegime), nil
}

var ErrSpecialTaxRateOverlap = errors.New("special tax rate overlaps another rate of the regime")

// SpecialTaxRegimeView is a legal regime as platform administrators manage it:
// its dated rates, how many companies are enrolled and whether the rates are
// about to run out.
type SpecialTaxRegimeView struct {
	domain.SpecialTaxRegime
	EnrolledProfiles int                     `json:"enrolledProfiles"`
	Status           domain.SpecialTaxStatus `json:"status"`
}
type SpecialTaxRegimesView struct {
	Items    []SpecialTaxRegimeView `json:"items"`
	Today    string                 `json:"today"`
	WarnDays int                    `json:"warnDays"`
}
type SpecialTaxRepository interface {
	SpecialTaxRegimes(ctx context.Context) ([]domain.SpecialTaxRegime, error)
	EnrolledProfiles(ctx context.Context) (map[string]int, error)
	BusinessDate(ctx context.Context, s scope) (string, error)
	CreateSpecialRate(ctx context.Context, s scope, regime string, in domain.SpecialTaxRateInput) (string, error)
	UpdateSpecialRate(ctx context.Context, id string, in domain.SpecialTaxRateInput) (int64, error)
	DeleteSpecialRate(ctx context.Context, id string) (int64, error)
}

// SpecialTaxes lets platform administrators keep the legal calendar of each
// special regime. A change applies to every enrolled company at once; issued
// documents keep the rate they were issued with.
type SpecialTaxes struct {
	repo  SpecialTaxRepository
	audit audit.Repository
}

func NewSpecialTaxes(repo SpecialTaxRepository, audit audit.Repository) *SpecialTaxes {
	return &SpecialTaxes{repo, audit}
}

func (a *SpecialTaxes) List(ctx context.Context, s scope) (SpecialTaxRegimesView, error) {
	out := SpecialTaxRegimesView{Items: []SpecialTaxRegimeView{}, WarnDays: domain.SpecialTaxWarnDays}
	regimes, err := a.repo.SpecialTaxRegimes(ctx)
	var enrolled map[string]int
	if err == nil {
		enrolled, err = a.repo.EnrolledProfiles(ctx)
	}
	if err == nil {
		out.Today, err = a.repo.BusinessDate(ctx, s)
	}
	if err != nil {
		return out, organizationFailure(apperror.Unavailable, "special_taxes_unavailable", "No pudimos cargar los impuestos.")
	}
	for _, regime := range regimes {
		out.Items = append(out.Items, SpecialTaxRegimeView{SpecialTaxRegime: regime, EnrolledProfiles: enrolled[regime.Code], Status: domain.SpecialTaxStatusOn(regime, out.Today)})
	}
	return out, nil
}

func (a *SpecialTaxes) CreateRate(ctx context.Context, s scope, regime string, in domain.SpecialTaxRateInput) (map[string]string, error) {
	if !domain.NormalizeSpecialTaxRateInput(&in) {
		return nil, invalidSpecialRate()
	}
	id, err := a.repo.CreateSpecialRate(ctx, s, regime, in)
	if err != nil {
		return nil, specialRateFailure(err, "No encontramos el IGV reducido.")
	}
	_ = a.audit.Record(ctx, s, "special_tax_rate.created", "special_tax_rate", id)
	return map[string]string{"id": id}, nil
}

func (a *SpecialTaxes) UpdateRate(ctx context.Context, s scope, id string, in domain.SpecialTaxRateInput) error {
	if !domain.NormalizeSpecialTaxRateInput(&in) {
		return invalidSpecialRate()
	}
	command, err := a.repo.UpdateSpecialRate(ctx, id, in)
	if err == nil && command == 0 {
		err = ErrNotFound
	}
	if err != nil {
		return specialRateFailure(err, "No encontramos el periodo.")
	}
	_ = a.audit.Record(ctx, s, "special_tax_rate.updated", "special_tax_rate", id)
	return nil
}

func (a *SpecialTaxes) DeleteRate(ctx context.Context, s scope, id string) error {
	command, err := a.repo.DeleteSpecialRate(ctx, id)
	if err == nil && command == 0 {
		err = ErrNotFound
	}
	if err != nil {
		return specialRateFailure(err, "No encontramos el periodo.")
	}
	_ = a.audit.Record(ctx, s, "special_tax_rate.deleted", "special_tax_rate", id)
	return nil
}

func invalidSpecialRate() error {
	return organizationFailure(apperror.Invalid, "invalid_special_tax_rate", "Revisa las fechas, el porcentaje (0 a 100 %) y la base legal.")
}

func specialRateFailure(err error, notFound string) error {
	switch {
	case errors.Is(err, ErrSpecialTaxRateOverlap):
		return organizationFailure(apperror.Conflict, "special_tax_rate_overlap", "Las fechas se cruzan con otro periodo de este IGV reducido.")
	case errors.Is(err, ErrNotFound):
		return organizationFailure(apperror.NotFound, "special_tax_rate_not_found", notFound)
	}
	return organizationFailure(apperror.Unavailable, "special_taxes_unavailable", "No pudimos guardar el periodo.")
}
