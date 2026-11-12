DELETE FROM staff_notices WHERE kind='delivery_unassigned';
ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account'));

DROP TABLE IF EXISTS delivery_partner_messages;
DROP TABLE IF EXISTS delivery_dispatch_offers;
DROP TABLE IF EXISTS delivery_dispatches;
DROP TABLE IF EXISTS location_delivery_partners;
DROP TABLE IF EXISTS location_delivery_settings;
DROP TABLE IF EXISTS delivery_partners;
