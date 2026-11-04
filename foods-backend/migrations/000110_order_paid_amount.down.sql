DROP INDEX IF EXISTS orders_pos_unpaid_idx;
DROP TRIGGER IF EXISTS payment_refunds_paid_amount ON payment_refunds;
DROP TRIGGER IF EXISTS payments_paid_amount ON payments;
DROP FUNCTION IF EXISTS payment_refunds_refresh_order_paid_amount();
DROP FUNCTION IF EXISTS payments_refresh_order_paid_amount();
DROP FUNCTION IF EXISTS refresh_order_paid_amount(uuid, uuid);
ALTER TABLE orders DROP COLUMN IF EXISTS paid_amount;
