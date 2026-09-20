DROP TRIGGER IF EXISTS locations_assign_default_fiscal_profile ON locations;
DROP FUNCTION IF EXISTS assign_default_fiscal_profile();
DROP TRIGGER IF EXISTS organizations_default_fiscal_profile ON organizations;
DROP FUNCTION IF EXISTS create_default_fiscal_profile();
DROP TABLE IF EXISTS organization_exchange_rates;
ALTER TABLE locations DROP CONSTRAINT IF EXISTS locations_fiscal_profile_scope_fk;
ALTER TABLE locations DROP COLUMN IF EXISTS fiscal_profile_id;
DROP TABLE IF EXISTS organization_fiscal_profiles;
