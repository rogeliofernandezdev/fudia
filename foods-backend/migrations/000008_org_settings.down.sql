ALTER TABLE organizations
  DROP COLUMN IF EXISTS currency_symbol,
  DROP COLUMN IF EXISTS currency_position,
  DROP COLUMN IF EXISTS currency_decimals,
  DROP COLUMN IF EXISTS tax_name,
  DROP COLUMN IF EXISTS tax_rate,
  DROP COLUMN IF EXISTS tax_included;
