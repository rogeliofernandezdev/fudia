-- The reduced IGV of Peru's MYPE restaurants is set by law, not by each
-- company: platform administrators register its dated rates and a fiscal
-- profile only says whether the company is enrolled. Replaces the per-profile
-- regimes of 000097.
DROP TABLE IF EXISTS fiscal_profile_tax_regime_components;
DROP TABLE IF EXISTS fiscal_profile_tax_regimes;

CREATE TABLE special_tax_regimes (
  code text PRIMARY KEY CHECK (code ~ '^[a-z][a-z0-9_]{0,39}$'),
  country_code char(2) NOT NULL CHECK (country_code ~ '^[A-Z]{2}$'),
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
  UNIQUE (code,country_code)
);

-- Total rate printed on the receipt (IGV + IPM), valid_to exclusive.
CREATE TABLE special_tax_rates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  regime_code text NOT NULL REFERENCES special_tax_regimes(code) ON DELETE CASCADE,
  valid_from date NOT NULL,
  valid_to date CHECK (valid_to IS NULL OR valid_to > valid_from),
  rate numeric(7,6) NOT NULL CHECK (rate BETWEEN 0 AND 1),
  legal_basis text NOT NULL CHECK (length(btrim(legal_basis)) BETWEEN 1 AND 120),
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT special_tax_rates_no_overlap EXCLUDE USING gist (
    regime_code WITH =,
    daterange(valid_from,valid_to,'[)') WITH &&
  )
);

INSERT INTO special_tax_regimes(code,country_code,name) VALUES
  ('pe_mype_restaurants','PE','MYPE restaurante (padrón SUNAT)');

-- Starting point: 2025 IGV 8 % + IPM 2 %; 2026 IGV 8 % + IPM 2,5 %;
-- 2027 IGV 12 % + IPM 3 %. Later extensions are registered from the platform.
INSERT INTO special_tax_rates(regime_code,valid_from,valid_to,rate,legal_basis) VALUES
  ('pe_mype_restaurants','2025-01-01','2026-01-01',0.100000,'Ley 32219'),
  ('pe_mype_restaurants','2026-01-01','2027-01-01',0.105000,'Leyes 32219 y 32387'),
  ('pe_mype_restaurants','2027-01-01','2028-01-01',0.150000,'Leyes 32219 y 32387');

ALTER TABLE organization_fiscal_profiles
  ADD COLUMN special_tax_regime text,
  ADD CONSTRAINT organization_fiscal_profiles_special_tax_regime_fk
    FOREIGN KEY (special_tax_regime,country_code)
    REFERENCES special_tax_regimes(code,country_code);
