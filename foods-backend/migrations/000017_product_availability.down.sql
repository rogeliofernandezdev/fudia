ALTER TABLE products
  DROP COLUMN IF EXISTS available_from,
  DROP COLUMN IF EXISTS available_until,
  DROP COLUMN IF EXISTS available_days,
  DROP COLUMN IF EXISTS available_until_time;
