ALTER TABLE organizations
  ADD COLUMN country_code char(2) NOT NULL DEFAULT 'PE'
  CHECK (country_code ~ '^[A-Z]{2}$');
