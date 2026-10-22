ALTER TABLE customer_addresses
  DROP CONSTRAINT IF EXISTS customer_addresses_point_check,
  DROP COLUMN IF EXISTS longitude,
  DROP COLUMN IF EXISTS latitude;
