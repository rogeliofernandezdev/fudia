package application

import (
	"context"
	"errors"
	audit "github.com/foods-platform/foods-backend/internal/modules/audit/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/apperror"
	"github.com/foods-platform/foods-backend/internal/platform/unitofwork"
)

var ErrTaxRegimeOverlap = errors.New("tax regime overlaps another regime of the profile")

// TaxProfile is the general tax of a fiscal profile, used when no regime applies.
type TaxProfile struct {
	General     domain.TaxComponent
	TaxIncluded bool
	Active      bool
}
type TaxRegimesView struct {
	Items       []domain.TaxRegime  `json:"items"`
	General     domain.TaxComponent `json:"general"`
	TaxIncluded bool                `json:"taxIncluded"`
	Effective   domain.EffectiveTax `json:"effective"`
}
type TaxRegimeRepository interface {
	// TaxProfile locks the profile row when lock is true so regime writes of the
	// same profile are serialized.
	TaxProfile(ctx context.Context, s scope, profileID string, lock bool) (TaxProfile, error)
	Regimes(ctx context.Context, s scope, profileID string) ([]domain.TaxRegime, error)
	CreateRegime(ctx context.Context, s scope, profileID string, in domain.TaxRegimeInput) (string, error)
	UpdateRegime(ctx context.Context, s scope, profileID, id string, in domain.TaxRegimeInput) (int64, error)
	DeleteRegime(ctx context.Context, s scope, profileID, id string) (int64, error)
	// BusinessDate is today in the session location's timezone.
	BusinessDate(ctx context.Context, s scope) (string, error)
}
type TaxRegimes struct {
	repo  TaxRegimeRepository
	work  unitofwork.Manager
	audit audit.Repository
}

func NewTaxRegimes(repo TaxRegimeRepository, work unitofwork.Manager, audit audit.Repository) *TaxRegimes {
	return &TaxRegimes{repo, work, audit}
}

func (a *TaxRegimes) List(ctx context.Context, s scope, profileID, date string) (TaxRegimesView, error) {
	var out TaxRegimesView
	if date != "" && !domain.ValidBusinessDate(date) {
		return out, organizationFailure(apperror.Invalid, "invalid_tax_date", "Revisa la fecha consultada.")
	}
	profile, err := a.repo.TaxProfile(ctx, s, profileID, false)
	if errors.Is(err, ErrNotFound) {
		return out, organizationFailure(apperror.NotFound, "profile_not_found", "No encontramos el perfil fiscal.")
	}
	if err != nil {
		return out, organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos cargar los regímenes de impuesto.")
	}
	items, err := a.repo.Regimes(ctx, s, profileID)
	if err == nil && date == "" {
		date, err = a.repo.BusinessDate(ctx, s)
	}
	if err != nil {
		return out, organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos cargar los regímenes de impuesto.")
	}
	return TaxRegimesView{
		Items:       items,
		General:     profile.General,
		TaxIncluded: profile.TaxIncluded,
		Effective:   domain.ResolveEffectiveTax(date, profile.General, profile.TaxIncluded, items),
	}, nil
}

// EffectiveTax resolves the tax that applies to a profile on a business date;
// an empty date means today in the session location's timezone.
func (a *TaxRegimes) EffectiveTax(ctx context.Context, s scope, profileID, date string) (domain.EffectiveTax, error) {
	view, err := a.List(ctx, s, profileID, date)
	return view.Effective, err
}

func (a *TaxRegimes) Create(ctx context.Context, s scope, profileID string, in domain.TaxRegimeInput) (domain.TaxRegime, error) {
	return a.save(ctx, s, profileID, "", in)
}
func (a *TaxRegimes) Update(ctx context.Context, s scope, profileID, id string, in domain.TaxRegimeInput) (domain.TaxRegime, error) {
	return a.save(ctx, s, profileID, id, in)
}

func (a *TaxRegimes) save(ctx context.Context, s scope, profileID, id string, in domain.TaxRegimeInput) (domain.TaxRegime, error) {
	var result domain.TaxRegime
	if !domain.NormalizeTaxRegimeInput(&in) {
		return result, organizationFailure(apperror.Invalid, "invalid_tax_regime", "Revisa el nombre, las fechas y los impuestos. La suma no puede superar el 100 %.")
	}
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return result, organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos guardar el régimen de impuesto.")
	}
	defer tx.Rollback(ctx)
	if err := a.writableProfile(tx.Context(), s, profileID); err != nil {
		return result, err
	}
	created := id == ""
	if created {
		id, err = a.repo.CreateRegime(tx.Context(), s, profileID, in)
	} else {
		var command int64
		command, err = a.repo.UpdateRegime(tx.Context(), s, profileID, id, in)
		if err == nil && command == 0 {
			err = ErrNotFound
		}
	}
	if errors.Is(err, ErrTaxRegimeOverlap) {
		return result, organizationFailure(apperror.Conflict, "tax_regime_overlap", "Las fechas se cruzan con otro régimen de este perfil fiscal.")
	}
	if errors.Is(err, ErrNotFound) {
		return result, organizationFailure(apperror.NotFound, "tax_regime_not_found", "No encontramos el régimen de impuesto.")
	}
	if err != nil || tx.Commit(ctx) != nil {
		return result, organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos guardar el régimen de impuesto.")
	}
	_ = a.audit.Record(ctx, s, map[bool]string{true: "tax_regime.created", false: "tax_regime.updated"}[created], "tax_regime", id)
	return domain.TaxRegime{ID: id, Name: in.Name, ValidFrom: in.ValidFrom, ValidTo: in.ValidTo, Components: in.Components, TotalRate: domain.SumTaxRates(in.Components)}, nil
}

func (a *TaxRegimes) Delete(ctx context.Context, s scope, profileID, id string) error {
	tx, err := a.work.Begin(ctx, unitofwork.Options{})
	if err != nil {
		return organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos eliminar el régimen de impuesto.")
	}
	defer tx.Rollback(ctx)
	if err := a.writableProfile(tx.Context(), s, profileID); err != nil {
		return err
	}
	command, err := a.repo.DeleteRegime(tx.Context(), s, profileID, id)
	if err == nil && command == 0 {
		return organizationFailure(apperror.NotFound, "tax_regime_not_found", "No encontramos el régimen de impuesto.")
	}
	if err != nil || tx.Commit(ctx) != nil {
		return organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos eliminar el régimen de impuesto.")
	}
	_ = a.audit.Record(ctx, s, "tax_regime.deleted", "tax_regime", id)
	return nil
}

func (a *TaxRegimes) writableProfile(ctx context.Context, s scope, profileID string) error {
	profile, err := a.repo.TaxProfile(ctx, s, profileID, true)
	if errors.Is(err, ErrNotFound) {
		return organizationFailure(apperror.NotFound, "profile_not_found", "No encontramos el perfil fiscal.")
	}
	if err != nil {
		return organizationFailure(apperror.Unavailable, "tax_regimes_unavailable", "No pudimos validar el perfil fiscal.")
	}
	if !profile.Active {
		return organizationFailure(apperror.Conflict, "profile_inactive", "Activa el perfil fiscal antes de cambiar sus impuestos.")
	}
	return nil
}
