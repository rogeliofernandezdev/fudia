ALTER TABLE organization_fiscal_profiles
  DROP CONSTRAINT IF EXISTS organization_fiscal_profiles_special_tax_regime_fk,
  DROP COLUMN IF EXISTS special_tax_regime;
DROP TABLE IF EXISTS special_tax_rates;
DROP TABLE IF EXISTS special_tax_regimes;

CREATE TABLE fiscal_profile_tax_regimes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  fiscal_profile_id uuid NOT NULL,
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  valid_from date NOT NULL,
  valid_to date CHECK (valid_to IS NULL OR valid_to > valid_from),
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id),
  FOREIGN KEY (fiscal_profile_id,organization_id)
    REFERENCES organization_fiscal_profiles(id,organization_id) ON DELETE CASCADE,
  CONSTRAINT fiscal_profile_tax_regimes_no_overlap EXCLUDE USING gist (
    fiscal_profile_id WITH =,
    daterange(valid_from,valid_to,'[)') WITH &&
  )
);

CREATE TABLE fiscal_profile_tax_regime_components (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  regime_id uuid NOT NULL,
  code text NOT NULL CHECK (code ~ '^[a-z][a-z0-9_]{0,19}$'),
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 30),
  rate numeric(7,6) NOT NULL CHECK (rate BETWEEN 0 AND 1),
  sort_order smallint NOT NULL CHECK (sort_order >= 0),
  UNIQUE (regime_id,code),
  FOREIGN KEY (regime_id,organization_id)
    REFERENCES fiscal_profile_tax_regimes(id,organization_id) ON DELETE CASCADE
);
