package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Taxes struct{ pool *pgxpool.Pool }

func NewTaxes(pool *pgxpool.Pool) *Taxes { return &Taxes{pool} }

var (
	_ application.TaxRepository        = (*Taxes)(nil)
	_ application.SpecialTaxRepository = (*Taxes)(nil)
)

func (r *Taxes) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

// specialTaxError maps a malformed id to not found, an unknown regime to not
// found and the no-overlap exclusion constraint to its domain error.
func specialTaxError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return application.ErrNotFound
	case errors.As(err, &pgErr) && (pgErr.Code == "22P02" || pgErr.Code == "23503"):
		return application.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23P01":
		return application.ErrSpecialTaxRateOverlap
	}
	return err
}

func (r *Taxes) TaxProfile(ctx context.Context, s scope, profileID string) (application.TaxProfile, error) {
	var out application.TaxProfile
	var regime *string
	err := r.db(ctx).QueryRow(ctx, `SELECT tax_name,tax_rate::text,tax_included,special_tax_regime FROM organization_fiscal_profiles WHERE id=$1 AND organization_id=$2`,
		profileID, s.OrganizationID).Scan(&out.General.Name, &out.General.Rate, &out.TaxIncluded, &regime)
	if err != nil {
		return out, specialTaxError(err)
	}
	out.General.Code = domain.GeneralTaxCode
	if regime != nil {
		regimes, err := specialTaxRegimes(ctx, r.db(ctx), *regime)
		if err != nil || len(regimes) == 0 {
			return out, errors.Join(err, errors.New("special tax regime not found"))
		}
		out.SpecialRegime = &regimes[0]
	}
	return out, nil
}

func (r *Taxes) BusinessDate(ctx context.Context, s scope) (string, error) {
	return businessDate(ctx, r.db(ctx), s)
}

func (r *Taxes) SpecialTaxRegimes(ctx context.Context) ([]domain.SpecialTaxRegime, error) {
	return specialTaxRegimes(ctx, r.db(ctx), "")
}

func (r *Taxes) EnrolledProfiles(ctx context.Context) (map[string]int, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT p.special_tax_regime,count(*)::int FROM organization_fiscal_profiles p
		JOIN organizations o ON o.id=p.organization_id AND o.active
		WHERE p.special_tax_regime IS NOT NULL AND p.active GROUP BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var code string
		var count int
		if err := rows.Scan(&code, &count); err != nil {
			return nil, err
		}
		out[code] = count
	}
	return out, rows.Err()
}

func (r *Taxes) CreateSpecialRate(ctx context.Context, s scope, regime string, in domain.SpecialTaxRateInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO special_tax_rates(regime_code,valid_from,valid_to,rate,legal_basis,created_by)
		VALUES($1,$2,$3,$4::numeric,$5,NULLIF($6,'')::uuid) RETURNING id`,
		regime, in.ValidFrom, in.ValidTo, in.Rate, in.LegalBasis, s.UserID).Scan(&id)
	return id, specialTaxError(err)
}

func (r *Taxes) UpdateSpecialRate(ctx context.Context, id string, in domain.SpecialTaxRateInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE special_tax_rates SET valid_from=$1,valid_to=$2,rate=$3::numeric,legal_basis=$4,updated_at=now() WHERE id=$5`,
		in.ValidFrom, in.ValidTo, in.Rate, in.LegalBasis, id)
	if err != nil {
		return 0, specialTaxError(err)
	}
	return tag.RowsAffected(), nil
}

func (r *Taxes) DeleteSpecialRate(ctx context.Context, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM special_tax_rates WHERE id=$1`, id)
	if errors.Is(specialTaxError(err), application.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// businessDate is today in the session location's timezone.
func businessDate(ctx context.Context, db database.Executor, s scope) (string, error) {
	var date string
	err := db.QueryRow(ctx, `
		SELECT (now() AT TIME ZONE COALESCE(
			(SELECT timezone FROM locations WHERE id=NULLIF($2,'')::uuid AND organization_id=$1),
			(SELECT timezone FROM organizations WHERE id=$1),
			'UTC'))::date::text`, s.OrganizationID, s.LocationID).Scan(&date)
	return date, err
}

// specialTaxRegimes reads the platform catalog with its dated rates; an empty
// code returns every regime.
func specialTaxRegimes(ctx context.Context, db database.Executor, code string) ([]domain.SpecialTaxRegime, error) {
	rows, err := db.Query(ctx, `
		SELECT g.code,g.country_code,g.name,
		       COALESCE(json_agg(json_build_object('id',t.id,'validFrom',t.valid_from::text,'validTo',t.valid_to::text,'rate',t.rate::text,'legalBasis',t.legal_basis) ORDER BY t.valid_from)
		                FILTER (WHERE t.id IS NOT NULL),'[]')
		FROM special_tax_regimes g
		LEFT JOIN special_tax_rates t ON t.regime_code=g.code
		WHERE $1='' OR g.code=$1
		GROUP BY g.code
		ORDER BY g.country_code,g.name`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SpecialTaxRegime{}
	for rows.Next() {
		var item domain.SpecialTaxRegime
		var rates []byte
		if err := rows.Scan(&item.Code, &item.Country, &item.Name, &rates); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rates, &item.Rates); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
