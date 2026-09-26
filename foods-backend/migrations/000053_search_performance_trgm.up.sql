CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS users_email_trgm_idx
  ON users USING gin (lower(email) gin_trgm_ops)
  WHERE active;

CREATE INDEX IF NOT EXISTS users_name_trgm_idx
  ON users USING gin (full_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS products_name_trgm_idx
  ON products USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS products_sku_trgm_idx
  ON products USING gin (sku gin_trgm_ops);

CREATE INDEX IF NOT EXISTS inventory_items_name_trgm_idx
  ON inventory_items USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS inventory_items_sku_trgm_idx
  ON inventory_items USING gin (sku gin_trgm_ops);

CREATE INDEX IF NOT EXISTS customers_name_trgm_idx
  ON customers USING gin (display_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS customers_code_trgm_idx
  ON customers USING gin (code gin_trgm_ops);

CREATE INDEX IF NOT EXISTS customers_phone_trgm_idx
  ON customers USING gin (phone gin_trgm_ops);

CREATE INDEX IF NOT EXISTS customers_email_trgm_idx
  ON customers USING gin (email gin_trgm_ops);

CREATE INDEX IF NOT EXISTS customers_document_trgm_idx
  ON customers USING gin (document_number gin_trgm_ops);

CREATE INDEX IF NOT EXISTS orders_code_trgm_idx
  ON orders USING gin (code gin_trgm_ops);

CREATE INDEX IF NOT EXISTS orders_customer_name_trgm_idx
  ON orders USING gin (customer_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS orders_customer_phone_trgm_idx
  ON orders USING gin (customer_phone gin_trgm_ops);

CREATE INDEX IF NOT EXISTS suppliers_name_trgm_idx
  ON suppliers USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS reservations_customer_name_trgm_idx
  ON reservations USING gin (customer_name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS reservations_customer_phone_trgm_idx
  ON reservations USING gin (customer_phone gin_trgm_ops);
