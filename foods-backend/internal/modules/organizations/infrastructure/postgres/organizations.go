package postgres

import (
	"context"
	"errors"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/application"
	"github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/platform/database"
	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
	"time"
)

type scope = requestscope.Scope
type Organizations struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Organizations { return &Organizations{pool} }
func (r *Organizations) db(ctx context.Context) database.Executor {
	return database.ExecutorFor(ctx, r.pool)
}
func organizationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

var _ application.OrganizationRepository = (*Organizations)(nil)

func (r *Organizations) Currencies(ctx context.Context) ([]domain.CurrencyOption, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT code,name,symbol,decimals FROM platform_currencies WHERE active ORDER BY name,code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.CurrencyOption{}
	for rows.Next() {
		var item domain.CurrencyOption
		if err := rows.Scan(&item.Code, &item.Name, &item.Symbol, &item.Decimals); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Organizations) Countries(ctx context.Context) ([]domain.CountryOption, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT c.code,c.name,c.default_currency
		FROM platform_countries c
		WHERE c.active
		ORDER BY c.name,c.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.CountryOption{}
	for rows.Next() {
		var item domain.CountryOption
		if err := rows.Scan(&item.Code, &item.Name, &item.DefaultCurrency); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Organizations) GlobalCountries(ctx context.Context) ([]domain.PlatformCountryOption, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT c.code,c.name,c.default_currency,COALESCE(c.calling_code,''),c.default_timezone
		FROM platform_countries c
		WHERE c.active
		ORDER BY c.name,c.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PlatformCountryOption{}
	for rows.Next() {
		var item domain.PlatformCountryOption
		if err := rows.Scan(&item.Code, &item.Name, &item.DefaultCurrency, &item.CallingCode, &item.DefaultTimezone); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Organizations) OnboardingCountries(ctx context.Context) ([]domain.PlatformCountryOption, error) {
	rows, err := r.db(ctx).Query(ctx, `
		SELECT c.code,c.name,c.default_currency,COALESCE(c.calling_code,''),c.default_timezone
		FROM platform_countries c
		WHERE c.active
		  AND EXISTS (
		    SELECT 1 FROM platform_whatsapp_channels w
		    WHERE w.country_code=c.code
		      AND w.active
		  )
		ORDER BY c.name,c.code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PlatformCountryOption{}
	for rows.Next() {
		var item domain.PlatformCountryOption
		if err := rows.Scan(&item.Code, &item.Name, &item.DefaultCurrency, &item.CallingCode, &item.DefaultTimezone); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (r *Organizations) CurrencyByCode(ctx context.Context, code string) (domain.CurrencyOption, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item domain.CurrencyOption
	err := r.db(ctx).QueryRow(ctx, `SELECT code,name,symbol,decimals FROM platform_currencies WHERE code=$1 AND active`, code).Scan(&item.Code, &item.Name, &item.Symbol, &item.Decimals)
	if err == nil {
		return item, true, nil
	}
	if err == pgx.ErrNoRows {
		return domain.CurrencyOption{}, false, nil
	}
	return domain.CurrencyOption{}, false, err
}
func (r *Organizations) CountryByCode(ctx context.Context, code string) (domain.CountryOption, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item domain.CountryOption
	err := r.db(ctx).QueryRow(ctx, `SELECT code,name,default_currency FROM platform_countries WHERE code=$1 AND active`, code).Scan(&item.Code, &item.Name, &item.DefaultCurrency)
	if err == nil {
		return item, true, nil
	}
	if err == pgx.ErrNoRows {
		return domain.CountryOption{}, false, nil
	}
	return domain.CountryOption{}, false, err
}
func (r *Organizations) OnboardingCountryByCode(ctx context.Context, code string) (domain.CountryOption, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	var item domain.CountryOption
	err := r.db(ctx).QueryRow(ctx, `
		SELECT c.code,c.name,c.default_currency
		FROM platform_countries c
		WHERE c.code=$1 AND c.active
		  AND EXISTS (
		    SELECT 1 FROM platform_whatsapp_channels w
		    WHERE w.country_code=c.code
		      AND w.active
		  )
	`, code).Scan(&item.Code, &item.Name, &item.DefaultCurrency)
	if err == nil {
		return item, true, nil
	}
	if err == pgx.ErrNoRows {
		return domain.CountryOption{}, false, nil
	}
	return domain.CountryOption{}, false, err
}
func (r *Organizations) Organization(ctx context.Context, s scope) (domain.OrganizationView, error) {
	var out domain.OrganizationView
	err := r.db(ctx).QueryRow(ctx, `SELECT id,legal_name,trade_name,tax_id,timezone,active FROM organizations WHERE id=$1`, s.OrganizationID).Scan(&out.ID, &out.LegalName, &out.TradeName, &out.TaxID, &out.Timezone, &out.Active)
	return out, organizationError(err)
}
func (r *Organizations) UpdateOrganization(ctx context.Context, s scope, in domain.OrganizationInput) (domain.OrganizationView, error) {
	var out domain.OrganizationView
	err := r.db(ctx).QueryRow(ctx, `UPDATE organizations SET legal_name=$1,trade_name=$2,tax_id=$3,timezone=$4,updated_at=now() WHERE id=$5 AND active RETURNING id,legal_name,trade_name,tax_id,timezone,active`, in.LegalName, in.TradeName, in.TaxID, in.Timezone, s.OrganizationID).Scan(&out.ID, &out.LegalName, &out.TradeName, &out.TaxID, &out.Timezone, &out.Active)
	return out, organizationError(err)
}
func (r *Organizations) CreateLocation(ctx context.Context, s scope, in domain.LocationInput) (domain.LocationView, error) {
	var out domain.LocationView
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,p.id FROM organization_fiscal_profiles p WHERE p.id=$10 AND p.organization_id=$1 AND p.active RETURNING id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id,(SELECT country_code FROM organization_fiscal_profiles WHERE id=fiscal_profile_id),(SELECT currency FROM organization_fiscal_profiles WHERE id=fiscal_profile_id),active`, s.OrganizationID, in.Name, in.Code, in.Address, in.Phone, in.OpeningHours, in.Latitude, in.Longitude, in.Timezone, in.FiscalProfileID).Scan(&out.ID, &out.Name, &out.Code, &out.Address, &out.Phone, &out.OpeningHours, &out.Latitude, &out.Longitude, &out.Timezone, &out.FiscalProfileID, &out.Country, &out.Currency, &out.Active)
	return out, organizationError(err)
}
func (r *Organizations) UpdateLocation(ctx context.Context, s scope, id string, in domain.LocationInput, active bool) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE locations l SET name=$1,code=$2,address=$3,phone=$4,opening_hours=$5,latitude=$6,longitude=$7,timezone=$8,fiscal_profile_id=p.id,active=$9,updated_at=now() FROM organization_fiscal_profiles p WHERE l.id=$10 AND l.organization_id=$11 AND p.id=$12 AND p.organization_id=l.organization_id AND p.active`, in.Name, in.Code, in.Address, in.Phone, in.OpeningHours, in.Latitude, in.Longitude, in.Timezone, active, id, s.OrganizationID, in.FiscalProfileID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) DeactivateLocation(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE locations SET active=false,updated_at=now() WHERE id=$1 AND organization_id=$2 AND active`, id, s.OrganizationID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) DeactivateProfile(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE organization_fiscal_profiles p SET active=false,updated_at=now() WHERE p.id=$1 AND p.organization_id=$2 AND NOT p.is_default AND NOT EXISTS(SELECT 1 FROM locations l WHERE l.fiscal_profile_id=p.id AND l.active)`, id, s.OrganizationID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) CreateRate(ctx context.Context, s scope, in domain.ExchangeRateInput, effective time.Time) (domain.ExchangeRate, error) {
	var out domain.ExchangeRate
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO organization_exchange_rates(organization_id,base_currency,quote_currency,rate,effective_at,source,provider_reference,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,base_currency,quote_currency,rate::text,to_char(effective_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),source,provider_reference`, s.OrganizationID, in.BaseCurrency, in.QuoteCurrency, in.Rate, effective, in.Source, in.ProviderReference, s.UserID).Scan(&out.ID, &out.BaseCurrency, &out.QuoteCurrency, &out.Rate, &out.EffectiveAt, &out.Source, &out.ProviderReference)
	return out, organizationError(err)
}
func (r *Organizations) DeleteRate(ctx context.Context, s scope, id string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `DELETE FROM organization_exchange_rates WHERE id=$1 AND organization_id=$2`, id, s.OrganizationID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) Settings(ctx context.Context, s scope) (domain.OrgSettings, error) {
	var out domain.OrgSettings
	err := r.db(ctx).QueryRow(ctx, `SELECT p.country_code,p.currency,p.currency_symbol,p.currency_position,p.currency_decimals,p.tax_name,p.tax_rate::float8,p.tax_included FROM organization_fiscal_profiles p JOIN organizations o ON o.id=p.organization_id WHERE p.organization_id=$1 AND p.is_default AND p.active AND o.active`, s.OrganizationID).Scan(&out.Country, &out.Currency, &out.CurrencySymbol, &out.CurrencyPosition, &out.CurrencyDecimals, &out.TaxName, &out.TaxRate, &out.TaxIncluded)
	return out, organizationError(err)
}
func (r *Organizations) LockDefaultProfile(ctx context.Context, s scope) (string, error) {
	var previousID string
	err := r.db(ctx).QueryRow(ctx, `SELECT id FROM organization_fiscal_profiles WHERE organization_id=$1 AND is_default FOR UPDATE`, s.OrganizationID).Scan(&previousID)
	return previousID, organizationError(err)
}
func (r *Organizations) ClearDefaultProfile(ctx context.Context, s scope) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE organization_fiscal_profiles SET is_default=false,updated_at=now() WHERE organization_id=$1 AND is_default`, s.OrganizationID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) SaveSettingsProfile(ctx context.Context, s scope, in domain.OrgSettingsInput, currency domain.CurrencyOption) (string, error) {
	var profileID string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO organization_fiscal_profiles(organization_id,country_code,currency,currency_symbol,currency_position,currency_decimals,tax_name,tax_rate,tax_included,is_default,active) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,true,true) ON CONFLICT(organization_id,country_code) DO UPDATE SET currency=EXCLUDED.currency,currency_symbol=EXCLUDED.currency_symbol,currency_position=EXCLUDED.currency_position,currency_decimals=EXCLUDED.currency_decimals,tax_name=EXCLUDED.tax_name,tax_rate=EXCLUDED.tax_rate,tax_included=EXCLUDED.tax_included,is_default=true,active=true,updated_at=now() RETURNING id`, s.OrganizationID, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded).Scan(&profileID)
	return profileID, organizationError(err)
}
func (r *Organizations) MigrateLocationProfile(ctx context.Context, s scope, profileID string, previousID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE locations SET fiscal_profile_id=$1 WHERE organization_id=$2 AND fiscal_profile_id=$3`, profileID, s.OrganizationID, previousID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) CompleteSetup(ctx context.Context, s scope) (time.Time, error) {
	var completedAt time.Time
	err := r.db(ctx).QueryRow(ctx, `
		UPDATE organization_operational_setup
		SET completed_at=COALESCE(completed_at,now()),completed_by=$2,updated_at=now()
		WHERE organization_id=$1
		RETURNING completed_at
	`, s.OrganizationID, s.UserID).Scan(&completedAt)
	return completedAt, organizationError(err)
}
func (r *Organizations) ClearProfileDefaults(ctx context.Context, s scope) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE organization_fiscal_profiles SET is_default=false,updated_at=now() WHERE organization_id=$1`, s.OrganizationID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) CreateProfile(ctx context.Context, s scope, in domain.FiscalProfileInput, currency domain.CurrencyOption) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO organization_fiscal_profiles(organization_id,country_code,currency,currency_symbol,currency_position,currency_decimals,tax_name,tax_rate,tax_included,is_default) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, s.OrganizationID, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, in.Default).Scan(&id)
	return id, organizationError(err)
}
func (r *Organizations) UpdateProfile(ctx context.Context, s scope, id string, in domain.FiscalProfileInput, currency domain.CurrencyOption) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `UPDATE organization_fiscal_profiles SET country_code=$1,currency=$2,currency_symbol=$3,currency_position=$4,currency_decimals=$5,tax_name=$6,tax_rate=$7,tax_included=$8,is_default=$9,active=true,updated_at=now() WHERE id=$10 AND organization_id=$11`, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, in.Default, id, s.OrganizationID)
	return tag.RowsAffected(), organizationError(err)
}
func (r *Organizations) DefaultProfileCount(ctx context.Context, s scope) (int, error) {
	var defaults int
	err := r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM organization_fiscal_profiles WHERE organization_id=$1 AND is_default`, s.OrganizationID).Scan(&defaults)
	return defaults, organizationError(err)
}
func scanFiscalProfile(rows pgx.Rows, item *domain.FiscalProfile) error {
	return rows.Scan(&item.ID, &item.Country, &item.Currency, &item.CurrencySymbol, &item.CurrencyPosition, &item.CurrencyDecimals, &item.TaxName, &item.TaxRate, &item.TaxIncluded, &item.Default, &item.Active, &item.AssignedLocations)
}
func (r *Organizations) Locations(ctx context.Context, s scope, page int, size int) (application.LocationsPage, error) {
	var out application.LocationsPage
	rows, err := r.db(ctx).Query(ctx, `SELECT l.id,l.name,l.code,l.address,l.phone,l.opening_hours,l.latitude,l.longitude,l.timezone,l.fiscal_profile_id,p.country_code,p.currency,l.active FROM locations l JOIN organization_fiscal_profiles p ON p.id=l.fiscal_profile_id AND p.organization_id=l.organization_id WHERE l.organization_id=$1 ORDER BY l.active DESC,l.name LIMIT $2 OFFSET $3`, s.OrganizationID, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.LocationView{}
	for rows.Next() {
		var item domain.LocationView
		if err = rows.Scan(&item.ID, &item.Name, &item.Code, &item.Address, &item.Phone, &item.OpeningHours, &item.Latitude, &item.Longitude, &item.Timezone, &item.FiscalProfileID, &item.Country, &item.Currency, &item.Active); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	var total int
	err = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM locations WHERE organization_id=$1`, s.OrganizationID).Scan(&total)
	if err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	out.Page = page
	out.PageSize = size
	return out, rows.Err()
}
func (r *Organizations) Rates(ctx context.Context, s scope, page int, size int) (application.RatesPage, error) {
	var out application.RatesPage
	rows, err := r.db(ctx).Query(ctx, `SELECT e.id,e.base_currency,e.quote_currency,e.rate::text,to_char(e.effective_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),e.source,e.provider_reference,u.full_name FROM organization_exchange_rates e LEFT JOIN users u ON u.id=e.created_by WHERE e.organization_id=$1 ORDER BY e.effective_at DESC LIMIT $2 OFFSET $3`, s.OrganizationID, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.ExchangeRate{}
	for rows.Next() {
		var item domain.ExchangeRate
		if err = rows.Scan(&item.ID, &item.BaseCurrency, &item.QuoteCurrency, &item.Rate, &item.EffectiveAt, &item.Source, &item.ProviderReference, &item.CreatedByName); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	var total int
	err = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM organization_exchange_rates WHERE organization_id=$1`, s.OrganizationID).Scan(&total)
	if err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	out.Page = page
	out.PageSize = size
	return out, rows.Err()
}
func (r *Organizations) Profiles(ctx context.Context, s scope, page int, size int) (application.ProfilesPage, error) {
	var out application.ProfilesPage
	rows, err := r.db(ctx).Query(ctx, `SELECT p.id,p.country_code,p.currency,p.currency_symbol,p.currency_position,p.currency_decimals,p.tax_name,p.tax_rate::text,p.tax_included,p.is_default,p.active,count(l.id) FROM organization_fiscal_profiles p LEFT JOIN locations l ON l.fiscal_profile_id=p.id AND l.organization_id=p.organization_id WHERE p.organization_id=$1 GROUP BY p.id ORDER BY p.is_default DESC,p.country_code LIMIT $2 OFFSET $3`, s.OrganizationID, size, (page-1)*size)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	items := []domain.FiscalProfile{}
	for rows.Next() {
		var item domain.FiscalProfile
		if err = scanFiscalProfile(rows, &item); err != nil {
			return out, err
		}
		items = append(items, item)
	}
	var total int
	err = r.db(ctx).QueryRow(ctx, `SELECT count(*) FROM organization_fiscal_profiles WHERE organization_id=$1`, s.OrganizationID).Scan(&total)
	if err != nil {
		return out, err
	}
	out.Items = items
	out.Total = total
	out.Page = page
	out.PageSize = size
	return out, rows.Err()
}
func (r *Organizations) Setup(ctx context.Context, s scope) (domain.RestaurantSetupView, error) {
	var out domain.RestaurantSetupView
	var serviceMode *string
	err := r.db(ctx).QueryRow(ctx, `
		SELECT ros.service_mode,ros.completed_at,
		  (SELECT count(*) FROM menu_categories mc WHERE mc.organization_id=$1 AND mc.active),
		  (SELECT count(*) FROM products p WHERE p.organization_id=$1 AND p.active AND p.category_id IS NOT NULL),
		  (SELECT count(*) FROM tables t WHERE t.organization_id=$1 AND t.location_id=$2 AND t.active),
		  (SELECT count(*) FROM cash_registers cr WHERE cr.organization_id=$1 AND cr.location_id=$2 AND cr.active),
		  (SELECT count(*) FROM users u WHERE u.organization_id=$1 AND u.active AND NOT u.platform_admin),
		  (SELECT count(*) FROM inventory_items ii WHERE ii.organization_id=$1 AND ii.active),
		  (SELECT count(*) FROM product_recipes pr WHERE pr.organization_id=$1 AND pr.active),
		  (SELECT count(*) FROM suppliers sp WHERE sp.organization_id=$1 AND sp.active)
		FROM organization_operational_setup ros
		WHERE ros.organization_id=$1
	`, s.OrganizationID, s.LocationID).Scan(
		&serviceMode, &out.CompletedAt,
		&out.Counts.Categories, &out.Counts.Products, &out.Counts.Tables,
		&out.Counts.CashRegisters, &out.Counts.Users, &out.Counts.InventoryItems,
		&out.Counts.Recipes, &out.Counts.Suppliers,
	)
	if serviceMode != nil {
		out.ServiceMode = *serviceMode
	}
	return out, organizationError(err)
}
func (r *Organizations) SetServiceMode(ctx context.Context, s scope, in application.ServiceModeInput) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE organization_operational_setup
		SET service_mode=$2,completed_at=NULL,completed_by=NULL,updated_at=now()
		WHERE organization_id=$1
	`, s.OrganizationID, in.ServiceMode)
	return tag.RowsAffected(), organizationError(err)
}
