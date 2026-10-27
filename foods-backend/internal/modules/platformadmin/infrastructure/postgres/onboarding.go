package postgres

import (
	"context"
	"encoding/json"
	billingcontracts "github.com/foods-platform/foods-backend/internal/modules/billing/contracts"
	identitycontracts "github.com/foods-platform/foods-backend/internal/modules/identity/contracts"
	orgcontracts "github.com/foods-platform/foods-backend/internal/modules/organizations/contracts"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/application"
	"github.com/foods-platform/foods-backend/internal/modules/platformadmin/domain"
	"github.com/jackc/pgx/v5"
)

// CreateTenant envía las cuatro inserciones base en un único pgx.Batch: una ida
// y vuelta a la base. No puede ser una sola sentencia con CTE porque el perfil
// fiscal lo crea un trigger de organizations y otra CTE no vería esa fila; en el
// batch cada sentencia sí ve la anterior y ubica la empresa por su tax_id único.
func (r *Platform) CreateTenant(ctx context.Context, s scope, in domain.TenantOnboardingInput, currency orgcontracts.CurrencyOption, hash string) (domain.TenantRecords, error) {
	var out domain.TenantRecords
	batch := &pgx.Batch{}
	batch.Queue(`INSERT INTO organizations(legal_name,trade_name,tax_id,timezone) VALUES($1,$2,$3,$4) RETURNING id`, in.LegalName, in.TradeName, in.TaxID, in.Timezone)
	batch.Queue(`
		UPDATE organization_fiscal_profiles
		SET country_code=$1,currency=$2,currency_symbol=$3,currency_position=$4,currency_decimals=$5,tax_name=$6,tax_rate=$7,tax_included=$8,updated_at=now()
		WHERE organization_id=(SELECT id FROM organizations WHERE tax_id=$9) AND is_default
		RETURNING id
	`, in.Country, currency.Code, currency.Symbol, in.CurrencyPosition, currency.Decimals, in.TaxName, in.TaxRate, in.TaxIncluded, in.TaxID)
	batch.Queue(`
		INSERT INTO locations(organization_id,name,code,address,phone,opening_hours,latitude,longitude,timezone,fiscal_profile_id)
		SELECT o.id,$2,$3,$4,$5,$6,$7,$8,$9,fp.id
		FROM organizations o
		JOIN organization_fiscal_profiles fp ON fp.organization_id=o.id AND fp.is_default
		WHERE o.tax_id=$1
		RETURNING id
	`, in.TaxID, in.LocationName, in.LocationCode, in.Address, in.LocationPhone, in.LocationHours, in.Latitude, in.Longitude, in.Timezone)
	batch.Queue(`
		INSERT INTO users(organization_id,email,full_name,password_hash)
		SELECT id,$2,$3,$4 FROM organizations WHERE tax_id=$1
		RETURNING id
	`, in.TaxID, in.AdminEmail, in.AdminName, hash)

	results := r.db(ctx).SendBatch(ctx, batch)
	targets := []*string{&out.OrganizationID, &out.FiscalProfileID, &out.LocationID, &out.AdministratorID}
	for i, target := range targets {
		if err := results.QueryRow().Scan(target); err != nil {
			_ = results.Close()
			return out, &application.TenantCreateError{Step: application.TenantCreateStep(i), Err: err}
		}
	}
	if err := results.Close(); err != nil {
		return out, &application.TenantCreateError{Step: application.TenantStepAdministrator, Err: err}
	}
	return out, nil
}
func (r *Platform) AssignAdministrator(ctx context.Context, s scope, userID string, roleID string, locationID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO user_roles(user_id,role_id,location_id) VALUES($1,$2,$3)`, userID, roleID, locationID)
	return tag.RowsAffected(), err
}
func (r *Platform) AuditOnboarded(ctx context.Context, s scope, organizationID string, locationID string, userID string) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `INSERT INTO audit_log(organization_id,location_id,user_id,action,entity_type,entity_id,metadata) VALUES($1,$2,$3,'organization.onboarded','organization',$1,jsonb_build_object('platformActor',$4::text))`, organizationID, locationID, userID, s.UserID)
	return tag.RowsAffected(), err
}

// SeedRoles inserta todos los roles predefinidos en una sola sentencia. Los que
// ya existen (reintento) se conservan y también devuelven su id, en el orden de seeds.
func (r *Platform) SeedRoles(ctx context.Context, organizationID string, seeds []identitycontracts.OrganizationRoleSeed) ([]string, error) {
	type roleRow struct {
		Name        string   `json:"name"`
		SystemKey   string   `json:"systemKey"`
		Description string   `json:"description"`
		MenuAccess  []string `json:"menuAccess"`
		Permissions []string `json:"permissions"`
	}
	payload := make([]roleRow, len(seeds))
	for i, seed := range seeds {
		payload[i] = roleRow{seed.Name, seed.SystemKey, seed.Description, nonNil(seed.MenuAccess), nonNil(seed.Permissions)}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	rows, err := r.db(ctx).Query(ctx, `
		WITH input AS MATERIALIZED (
			SELECT e.item->>'name' AS name, e.item->>'systemKey' AS system_key, e.item->>'description' AS description,
			       ARRAY(SELECT jsonb_array_elements_text(e.item->'menuAccess')) AS menu_access,
			       ARRAY(SELECT jsonb_array_elements_text(e.item->'permissions')) AS permissions,
			       e.position
			FROM jsonb_array_elements($2::jsonb) WITH ORDINALITY AS e(item,position)
		), inserted AS (
			INSERT INTO roles(organization_id,name,system_key,description,menu_access,permissions,active)
			SELECT $1,name,system_key,description,menu_access,permissions,true
			FROM input ORDER BY position
			ON CONFLICT(organization_id,name) DO NOTHING
			RETURNING id,name
		)
		SELECT COALESCE(n.id,existing.id)
		FROM input i
		LEFT JOIN inserted n ON n.name=i.name
		LEFT JOIN roles existing ON existing.organization_id=$1 AND existing.name=i.name
		ORDER BY i.position
	`, organizationID, raw)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
func (r *Platform) SeedPaymentMethods(ctx context.Context, organizationID string) error {
	if _, err := r.db(ctx).Exec(ctx, `SELECT 1 FROM organizations WHERE id=$1 FOR UPDATE`, organizationID); err != nil {
		return err
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO payment_methods(
			organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
		)
		SELECT $1,t.code,t.name,t.description,t.active,t.sales_enabled,t.expenses_enabled,t.affects_cash,t.sort_order
		FROM payment_method_templates t
		-- Country templates (Peru: cash, card, Yape, Plin) or the general set.
		WHERE t.active AND t.country_code=CASE WHEN (
		    SELECT fp.country_code::text FROM organization_fiscal_profiles fp
		    WHERE fp.organization_id=$1 ORDER BY fp.is_default DESC,fp.created_at LIMIT 1
		  )='PE' THEN 'PE' ELSE '' END
		AND NOT EXISTS(SELECT 1 FROM payment_methods p WHERE p.organization_id=$1 AND p.code=t.code)
		ORDER BY t.sort_order,t.code
		LIMIT GREATEST($2-(SELECT count(*) FROM payment_methods WHERE organization_id=$1),0)
		ON CONFLICT (organization_id,code) DO NOTHING
	`, organizationID, billingcontracts.MaxPaymentMethods)
	return err
}

// SeedOperationalDefaults precarga la operación inicial dentro de la transacción
// del onboarding. Cada tipo de registro se inserta con una sola sentencia y todas
// viajan en un único pgx.Batch: una ida y vuelta a la base. Cada sentencia es
// idempotente (restricciones únicas con ON CONFLICT DO NOTHING), así un reintento
// del paso no duplica registros ni sobrescribe la modalidad ya elegida.
func (r *Platform) SeedOperationalDefaults(ctx context.Context, organizationID string, locationID string, userID string, defaults domain.TenantOperationalDefaults) error {
	batch := &pgx.Batch{}
	batch.Queue(`
		INSERT INTO expense_categories(organization_id,name,active)
		SELECT $1,name,active
		FROM expense_category_templates
		ON CONFLICT DO NOTHING
	`, organizationID)
	// Zone templates plus the zone the default tables need, in case the
	// templates ever stop providing it.
	batch.Queue(`
		INSERT INTO zones(organization_id,location_id,name,sort_order,active)
		SELECT $1::uuid,$2::uuid,name,sort_order,active
		FROM location_zone_templates
		UNION ALL
		SELECT $1::uuid,$2::uuid,$3::text,0,true
		WHERE NOT EXISTS(SELECT 1 FROM location_zone_templates WHERE name=$3::text)
		ON CONFLICT DO NOTHING
	`, organizationID, locationID, defaults.TableZone)
	batch.Queue(`
		INSERT INTO cash_registers(organization_id,location_id,name,blind_close,active,created_by)
		SELECT $1,$2,name,blind_close,active,$3
		FROM cash_register_templates
		ON CONFLICT DO NOTHING
	`, organizationID, locationID, userID)
	batch.Queue(`
		INSERT INTO menu_categories(organization_id,name,sort_order,active,product_scope)
		VALUES($1,$2,0,true,$3)
		ON CONFLICT (organization_id,name) DO NOTHING
	`, organizationID, defaults.ProductCategory, defaults.ProductCategoryScope)
	batch.Queue(`
		INSERT INTO tables(organization_id,location_id,name,seats,zone,active,qr_token,qr_enabled)
		SELECT $1,$2,t.name,t.seats,$3,true,replace(gen_random_uuid()::text,'-',''),t.qr_enabled
		FROM unnest($4::text[],$5::integer[],$6::boolean[]) WITH ORDINALITY AS t(name,seats,qr_enabled,position)
		ORDER BY t.position
		ON CONFLICT (organization_id,location_id,name) DO NOTHING
	`, organizationID, locationID, defaults.TableZone, defaults.TableNames(), defaults.TableSeats(), defaults.TableQREnabled())
	batch.Queue(`
		INSERT INTO organization_operational_setup(organization_id,service_mode)
		VALUES($1,$2)
		ON CONFLICT (organization_id) DO NOTHING
	`, organizationID, defaults.ServiceMode)

	results := r.db(ctx).SendBatch(ctx, batch)
	for range batch.Len() {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}
	return results.Close()
}
