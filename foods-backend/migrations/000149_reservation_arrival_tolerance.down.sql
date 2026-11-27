DELETE FROM staff_notices WHERE kind='reservation_no_show';
ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account','delivery_unassigned','delivery_pickup_delayed','pay_detected'));

DROP INDEX IF EXISTS reservations_open_due_idx;
ALTER TABLE reservations
  DROP CONSTRAINT IF EXISTS reservations_expired_no_show_check,
  DROP COLUMN IF EXISTS expired_at,
  DROP COLUMN IF EXISTS deposit_amount,
  DROP COLUMN IF EXISTS tolerance_minutes;
ALTER TABLE locations
  DROP COLUMN IF EXISTS reservation_deposit_amount,
  DROP COLUMN IF EXISTS reservation_tolerance_minutes;
