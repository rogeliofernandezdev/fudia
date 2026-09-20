CREATE TABLE organization_fiscal_profiles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  country_code char(2) NOT NULL CHECK (country_code ~ '^[A-Z]{2}$'),
  currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  currency_symbol text NOT NULL,
  currency_position text NOT NULL CHECK (currency_position IN ('before','after')),
  currency_decimals smallint NOT NULL CHECK (currency_decimals BETWEEN 0 AND 4),
  tax_name text NOT NULL CHECK (length(btrim(tax_name)) BETWEEN 1 AND 30),
  tax_rate numeric(7,6) NOT NULL CHECK (tax_rate BETWEEN 0 AND 1),
  tax_included boolean NOT NULL DEFAULT false,
  is_default boolean NOT NULL DEFAULT false,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id,country_code),
  UNIQUE (id,organization_id)
);

CREATE UNIQUE INDEX organization_fiscal_profiles_one_default_idx
  ON organization_fiscal_profiles(organization_id) WHERE is_default;

INSERT INTO organization_fiscal_profiles (
  organization_id,country_code,currency,currency_symbol,currency_position,
  currency_decimals,tax_name,tax_rate,tax_included,is_default
)
SELECT id,country_code,currency,currency_symbol,currency_position,
       currency_decimals,tax_name,tax_rate,tax_included,true
FROM organizations;

ALTER TABLE locations ADD COLUMN fiscal_profile_id uuid;

UPDATE locations l
SET fiscal_profile_id = p.id
FROM organization_fiscal_profiles p
WHERE p.organization_id=l.organization_id AND p.is_default;

ALTER TABLE locations
  ALTER COLUMN fiscal_profile_id SET NOT NULL,
  ADD CONSTRAINT locations_fiscal_profile_scope_fk
    FOREIGN KEY (fiscal_profile_id,organization_id)
    REFERENCES organization_fiscal_profiles(id,organization_id);

CREATE TABLE organization_exchange_rates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  base_currency char(3) NOT NULL CHECK (base_currency ~ '^[A-Z]{3}$'),
  quote_currency char(3) NOT NULL CHECK (quote_currency ~ '^[A-Z]{3}$'),
  rate numeric(20,10) NOT NULL CHECK (rate > 0),
  effective_at timestamptz NOT NULL,
  source text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','provider')),
  provider_reference text,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (base_currency <> quote_currency),
  UNIQUE (organization_id,base_currency,quote_currency,effective_at)
);

CREATE INDEX organization_exchange_rates_lookup_idx
  ON organization_exchange_rates(organization_id,base_currency,quote_currency,effective_at DESC);

CREATE FUNCTION create_default_fiscal_profile() RETURNS trigger AS $$
BEGIN
  INSERT INTO organization_fiscal_profiles (
    organization_id,country_code,currency,currency_symbol,currency_position,
    currency_decimals,tax_name,tax_rate,tax_included,is_default
  ) VALUES (NEW.id,'PE','PEN','S/','before',2,'IGV',0.180000,false,true);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER organizations_default_fiscal_profile
AFTER INSERT ON organizations
FOR EACH ROW EXECUTE FUNCTION create_default_fiscal_profile();

CREATE FUNCTION assign_default_fiscal_profile() RETURNS trigger AS $$
BEGIN
  IF NEW.fiscal_profile_id IS NULL THEN
    SELECT id INTO NEW.fiscal_profile_id
    FROM organization_fiscal_profiles
    WHERE organization_id=NEW.organization_id AND is_default AND active;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER locations_assign_default_fiscal_profile
BEFORE INSERT ON locations
FOR EACH ROW EXECUTE FUNCTION assign_default_fiscal_profile();
