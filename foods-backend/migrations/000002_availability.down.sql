ALTER TABLE products DROP CONSTRAINT IF EXISTS products_quota_requires_manual;
ALTER TABLE products DROP COLUMN IF EXISTS default_daily_quota;
ALTER TABLE products DROP COLUMN IF EXISTS stock_mode;
ALTER TABLE organizations DROP COLUMN IF EXISTS inventory_mode;
