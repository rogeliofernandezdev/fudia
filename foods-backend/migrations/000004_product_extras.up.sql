ALTER TABLE products ADD COLUMN IF NOT EXISTS prep_minutes integer;
ALTER TABLE products ADD COLUMN IF NOT EXISTS allergens text[] DEFAULT '{}';
ALTER TABLE products ADD COLUMN IF NOT EXISTS featured boolean NOT NULL DEFAULT false;
ALTER TABLE products ADD COLUMN IF NOT EXISTS cost_price numeric(12,2);
