DELETE FROM customer_notices WHERE kind='order_sent';
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_order_kind_dedupe_key;
ALTER TABLE customer_notices DROP COLUMN dedupe_key;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_order_id_kind_key UNIQUE (order_id,kind);
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_kind_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_kind_check CHECK (kind IN ('order_ready'));
ALTER TABLE concierge_order_requests DROP COLUMN round_id, DROP COLUMN customer_phone;
