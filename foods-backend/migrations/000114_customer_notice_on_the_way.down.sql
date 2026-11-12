DELETE FROM customer_notices WHERE kind='order_on_the_way';
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_kind_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_kind_check
  CHECK (kind IN ('order_ready','order_sent'));
