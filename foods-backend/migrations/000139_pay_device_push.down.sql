DROP INDEX IF EXISTS pay_devices_push_location_idx;
ALTER TABLE pay_devices
  DROP COLUMN IF EXISTS woken_at,
  DROP COLUMN IF EXISTS push_token_updated_at,
  DROP COLUMN IF EXISTS push_token;
