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

type TaxRegimes struct{ pool *pgxpool.Pool }

func NewTaxRegimes(pool *pgxpool.Pool) *TaxRegimes { return &TaxRegimes{pool} }

var _ application.TaxRegimeRepository = (*TaxRegimes)(nil)

func (r *TaxRegimes) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}

// taxRegimeError maps a malformed id to not found and the no-overlap
// exclusion constraint to its domain error.
func taxRegimeError(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return application.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "22P02":
		return application.ErrNotFound
	case errors.As(err, &pgErr) && pgErr.Code == "23P01":
		return application.ErrTaxRegimeOverlap
	}
	return err
}

func (r *TaxRegimes) TaxProfile(ctx context.Context, s scope, profileID string, lock bool) (application.TaxProfile, error) {
	var out application.TaxProfile
	query := `SELECT tax_name,tax_rate::text,tax_included,active FROM organization_fiscal_profiles WHERE id=$1 AND organization_id=$2`
	if lock {
		query += ` FOR UPDATE`
	}
	err := r.db(ctx).QueryRow(ctx, query, profileID, s.OrganizationID).Scan(&out.General.Name, &out.General.Rate, &out.TaxIncluded, &out.Active)
	out.General.Code = domain.GeneralTaxCode
	return out, taxRegimeError(err)
}

func (r *TaxRegimes) Regimes(ctx context.Context, s scope, profileID string) ([]domain.TaxRegime, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT r.id,r.name,r.valid_from::text,r.valid_to::text,
		       COALESCE(json_agg(json_build_object('code',c.code,'name',c.name,'rate',c.rate::text) ORDER BY c.sort_order)
		                FILTER (WHERE c.id IS NOT NULL),'[]')
		FROM fiscal_profile_tax_regimes r
		LEFT JOIN fiscal_profile_tax_regime_components c ON c.regime_id=r.id AND c.organization_id=r.organization_id
		WHERE r.organization_id=$1 AND r.fiscal_profile_id=$2
		GROUP BY r.id
		ORDER BY r.valid_from DESC`, s.OrganizationID, profileID)
	if err != nil {
		return nil, taxRegimeError(err)
	}
	defer rows.Close()
	items := []domain.TaxRegime{}
	for rows.Next() {
		var item domain.TaxRegime
		var components []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.ValidFrom, &item.ValidTo, &components); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(components, &item.Components); err != nil {
			return nil, err
		}
		item.TotalRate = domain.SumTaxRates(item.Components)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *TaxRegimes) CreateRegime(ctx context.Context, s scope, profileID string, in domain.TaxRegimeInput) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
		INSERT INTO fiscal_profile_tax_regimes(organization_id,fiscal_profile_id,name,valid_from,valid_to,created_by)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid)
		RETURNING id`, s.OrganizationID, profileID, in.Name, in.ValidFrom, in.ValidTo, s.UserID).Scan(&id)
	if err != nil {
		return "", taxRegimeError(err)
	}
	return id, r.insertComponents(ctx, s, id, in.Components)
}

func (r *TaxRegimes) UpdateRegime(ctx context.Context, s scope, profileID, id string, in domain.TaxRegimeInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE fiscal_profile_tax_regimes SET name=$1,valid_from=$2,valid_to=$3,updated_at=now()
		WHERE id=$4 AND organization_id=$5 AND fiscal_profile_id=$6`, in.Name, in.ValidFrom, in.ValidTo, id, s.OrganizationID, profileID)
	if err != nil || tag.RowsAffected() == 0 {
		return 0, taxRegimeError(err)
	}
	if _, err = r.db(ctx).Exec(ctx, `DELETE FROM fiscal_profile_tax_regime_components WHERE regime_id=$1 AND organization_id=$2`, id, s.OrganizationID); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), r.insertComponents(ctx, s, id, in.Components)
}

func (r *TaxRegimes) insertComponents(ctx context.Context, s scope, regimeID string, components []domain.TaxComponent) error {
	codes, names, rates := make([]string, len(components)), make([]string, len(components)), make([]string, len(components))
	for i, c := range components {
		codes[i], names[i], rates[i] = c.Code, c.Name, c.Rate
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO fiscal_profile_tax_regime_components(organization_id,regime_id,code,name,rate,sort_order)
		SELECT $1,$2,c.code,c.name,c.rate::numeric,(c.ord-1)::smallint
		FROM unnest($3::text[],$4::text[],$5::text[]) WITH ORDINALITY AS c(code,name,rate,ord)`,
		s.OrganizationID, regimeID, codes, names, rates)
	return err
}

func (r *TaxRegimes) DeleteRegime(ctx context.Context, s scope, profileID, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM fiscal_profile_tax_regimes WHERE id=$1 AND organization_id=$2 AND fiscal_profile_id=$3`, id, s.OrganizationID, profileID)
	if err != nil {
		if errors.Is(taxRegimeError(err), application.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *TaxRegimes) BusinessDate(ctx context.Context, s scope) (string, error) {
	var date string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT (now() AT TIME ZONE COALESCE(
			(SELECT timezone FROM locations WHERE id=NULLIF($2,'')::uuid AND organization_id=$1),
			(SELECT timezone FROM organizations WHERE id=$1),
			'UTC'))::date::text`, s.OrganizationID, s.LocationID).Scan(&date)
	return date, err
}
