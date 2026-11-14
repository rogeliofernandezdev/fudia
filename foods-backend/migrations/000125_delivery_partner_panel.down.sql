DROP INDEX IF EXISTS delivery_dispatch_offers_partner_idx;

-- Un reparto entregado vuelve a «recogido», el último estado que conocía el esquema anterior.
UPDATE delivery_dispatches SET status='picked_up' WHERE status='delivered';
ALTER TABLE delivery_dispatches DROP CONSTRAINT delivery_dispatches_check;
ALTER TABLE delivery_dispatches ADD CONSTRAINT delivery_dispatches_check
  CHECK (status NOT IN ('accepted','picked_up') OR accepted_partner_id IS NOT NULL);
ALTER TABLE delivery_dispatches DROP CONSTRAINT delivery_dispatches_status_check;
ALTER TABLE delivery_dispatches ADD CONSTRAINT delivery_dispatches_status_check
  CHECK (status IN ('searching','accepted','unassigned','picked_up','own','cancelled'));

ALTER TABLE delivery_dispatches
  DROP CONSTRAINT delivery_dispatches_courier_location_check,
  DROP COLUMN courier_located_at,
  DROP COLUMN courier_longitude,
  DROP COLUMN courier_latitude,
  DROP COLUMN delivered_at;

DROP INDEX IF EXISTS delivery_partners_panel_token_idx;
ALTER TABLE delivery_partners DROP COLUMN panel_token;
