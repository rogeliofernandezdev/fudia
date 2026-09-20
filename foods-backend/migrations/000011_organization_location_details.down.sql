ALTER TABLE locations
  DROP COLUMN IF EXISTS updated_at,
  DROP COLUMN IF EXISTS timezone;

ALTER TABLE organizations ALTER COLUMN tax_id TYPE varchar(11);
