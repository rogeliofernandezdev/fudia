DROP INDEX IF EXISTS purchase_orders_status_idx;
DROP TABLE IF EXISTS purchase_order_items;

ALTER TABLE purchase_orders
  DROP COLUMN IF EXISTS cancelled_at,
  DROP COLUMN IF EXISTS received_at,
  DROP COLUMN IF EXISTS approved_at,
  DROP COLUMN IF EXISTS updated_at,
  DROP COLUMN IF EXISTS expected_at,
  DROP COLUMN IF EXISTS notes,
  DROP CONSTRAINT IF EXISTS purchase_orders_scope_uq;
