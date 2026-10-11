package postgres

import (
	"context"
	"errors"
	identitydomain "github.com/foods-platform/foods-backend/internal/modules/identity/domain"
	orgdomain "github.com/foods-platform/foods-backend/internal/modules/organizations/domain"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/jackc/pgx/v5"
)

func (r *Platform) CreateOrganization(ctx context.Context, s scope, in domain.TenantOnboardingInput) (string, error) {
	var organizationID string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO organizations(legal_name,trade_name,tax_id,timezone) VALUES($1,$2,$3,$4) RETURNING id`, in.LegalName, in.TradeName, in.TaxID, in.Timezone).Scan(&organizationID)
	return organizationID, err
}
func (r *Platform) ConfigureFiscal(ctx context.Context, s scope, organizationID string, in domain.TenantOnboardingInput, currency orgdomain.CurrencyOption) (string, error) {
	var profileID string
	err := r.db(ctx).QueryRow(ctx, `UPDATE organization_fiscal_profiles SET country_code=$1,currency=$2,currency_symbol=$3,currency_position=$4,currency_decimals=$5,tax_name=$6,tax_rate=$7,tax_included=$8,updated_at=now() WHERE organization_id=$9 AND is_default RETURNING id`, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, organizationID).Scan(&profileID)
	return profileID, err
}
func (r *Platform) CreateLocation(ctx context.Context, s scope, organizationID string, profileID string, in domain.TenantOnboardingInput) (string, error) {
	var locationID string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO locations(organization_id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, organizationID, in.LocationName, in.LocationCode, in.Address, in.LocationPhone, in.LocationHours, in.Latitude, in.Longitude, in.Timezone, profileID).Scan(&locationID)
	return locationID, err
}
func (r *Platform) CreateAdministrator(ctx context.Context, s scope, organizationID string, in domain.TenantOnboardingInput, hash string) (string, error) {
	var userID string
	err := r.db(ctx).QueryRow(ctx, `INSERT INTO users(organization_id,email,full_name,password_hash) VALUES($1,$2,$3,$4) RETURNING id`, organizationID, in.AdminEmail, in.AdminName, string(hash)).Scan(&userID)
	return userID, err
}
func (r *Platform) CreateOperationalSetup(ctx context.Context, s scope, organizationID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO organization_operational_setup(organization_id) VALUES($1)`, organizationID)
	return tag.RowsAffected(), err
}
func (r *Platform) AssignAdministrator(ctx context.Context, s scope, userID string, roleID string, locationID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, userID, roleID, locationID)
	return tag.RowsAffected(), err
}
func (r *Platform) AuditOnboarded(ctx context.Context, s scope, organizationID string, locationID string, userID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'organization.onboarded','organization',$1,jsonb_build_object('platformActor',$4::text))`, organizationID, locationID, userID, s.UserID)
	return tag.RowsAffected(), err
}
func (r *Platform) SeedRole(ctx context.Context, organizationID string, seed identitydomain.OrganizationRoleSeed) (string, error) {
	var id string
	err := r.db(ctx).QueryRow(ctx, `
			INSERT INTO roles(organization_id,name,system_key,description,menu_access,permissions,active)
			VALUES($1,$2,$3,$4,$5,$6,true)
			ON CONFLICT(organization_id,name) DO NOTHING
			RETURNING id
		`, organizationID, seed.Name, seed.SystemKey, seed.Description, seed.MenuAccess, seed.Permissions).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db(ctx).QueryRow(ctx, `SELECT id FROM roles WHERE organization_id=$1 AND name=$2`, organizationID, seed.Name).Scan(&id)
	}
	return id, err
}
func (r *Platform) SeedPaymentMethods(ctx context.Context, organizationID string) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO payment_methods(
			organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		)
		SELECT $1,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		FROM payment_method_templates
		ON CONFLICT (organization_id,code) DO NOTHING
	`, organizationID)
	return err
}
func (r *Platform) SeedOperationalDefaults(ctx context.Context, organizationID string, locationID string, userID string) error {
	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO expense_categories(organization_id,name,active)
		SELECT $1,name,active
		FROM expense_category_templates
		ON CONFLICT DO NOTHING
	`, organizationID); err != nil {
		return err
	}

	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO zones(organization_id,location_id,name,sort_order,active)
		SELECT $1,$2,name,sort_order,active
		FROM location_zone_templates
		ON CONFLICT DO NOTHING
	`, organizationID, locationID); err != nil {
		return err
	}

	if _, err := r.db(ctx).Exec(ctx, `
		INSERT INTO cash_registers(organization_id,location_id,name,blind_close,active,created_by)
		SELECT $1,$2,name,blind_close,active,$3
		FROM cash_register_templates
		ON CONFLICT DO NOTHING
	`, organizationID, locationID, userID); err != nil {
		return err
	}

	return nil
}
