DELETE FROM customer_notices WHERE kind='delivery_contact_request';
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_kind_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_kind_check
  CHECK (kind IN ('order_ready','order_sent','order_on_the_way'));

DELETE FROM staff_notices WHERE kind='delivery_pickup_delayed';
ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account','delivery_unassigned'));

DROP TABLE IF EXISTS delivery_failed_deliveries;

ALTER TABLE delivery_dispatch_offers DROP COLUMN released_at;

DROP INDEX IF EXISTS delivery_dispatches_courier_token_idx;
ALTER TABLE delivery_dispatches
  DROP CONSTRAINT delivery_dispatches_courier_check,
  DROP COLUMN delivery_issue_distance_m,
  DROP COLUMN delivery_issue_reported_at,
  DROP COLUMN pickup_alerted_at,
  DROP COLUMN pickup_alert_ack_level,
  DROP COLUMN courier_assigned_at,
  DROP COLUMN courier_token,
  DROP COLUMN courier_name;
