ALTER TABLE organizations
  ADD COLUMN currency_symbol text NOT NULL DEFAULT 'S/',
  ADD COLUMN currency_position text NOT NULL DEFAULT 'before' CHECK (currency_position IN ('before','after')),
  ADD COLUMN currency_decimals smallint NOT NULL DEFAULT 2 CHECK (currency_decimals BETWEEN 0 AND 4),
  ADD COLUMN tax_name text NOT NULL DEFAULT 'IGV',
  ADD COLUMN tax_rate numeric(5,4) NOT NULL DEFAULT 0.1800 CHECK (tax_rate BETWEEN 0 AND 1),
  ADD COLUMN tax_included boolean NOT NULL DEFAULT false;
