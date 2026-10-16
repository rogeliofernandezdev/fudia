ALTER TABLE orders
  DROP CONSTRAINT IF EXISTS orders_delivery_point_check,
  DROP COLUMN IF EXISTS delivery_distance_km,
  DROP COLUMN IF EXISTS delivery_longitude,
  DROP COLUMN IF EXISTS delivery_latitude;
DROP TABLE IF EXISTS concierge_delivery_requests;
DROP TABLE IF EXISTS delivery_fee_tiers;
