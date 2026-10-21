ALTER TABLE order_items
  DROP CONSTRAINT IF EXISTS order_items_product_required;

DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS inventory_entries;

ALTER TABLE stock_balances
  DROP CONSTRAINT IF EXISTS stock_balances_non_negative;

DROP INDEX IF EXISTS inventory_items_product_unique_idx;
ALTER TABLE inventory_items
  DROP CONSTRAINT IF EXISTS inventory_items_product_scope_fk,
  DROP COLUMN IF EXISTS product_id;

ALTER TABLE product_availability
  RENAME COLUMN portion_quantity TO daily_quota;

ALTER TABLE products
  ADD COLUMN stock_mode text NOT NULL DEFAULT 'none'
    CHECK (stock_mode IN ('none','manual','linked','recipe')),
  ADD COLUMN default_daily_quota integer
    CHECK (default_daily_quota IS NULL OR default_daily_quota >= 0);

UPDATE products
SET stock_mode = CASE quantity_control
  WHEN 'portions' THEN 'manual'
  WHEN 'inventory' THEN 'linked'
  ELSE 'none'
END;

ALTER TABLE products
  ADD CONSTRAINT products_quota_requires_manual
    CHECK (stock_mode = 'manual' OR default_daily_quota IS NULL);

ALTER TABLE products DROP COLUMN quantity_control;
