DROP TABLE order_service_items;
DROP INDEX order_round_request_key_idx;
ALTER TABLE order_kitchen_rounds DROP COLUMN request_key;
ALTER TABLE orders DROP CONSTRAINT orders_bill_closed_by_tenant_fk;
ALTER TABLE orders DROP COLUMN bill_closed_by, DROP COLUMN bill_closed_at;
ALTER TABLE products DROP COLUMN service_destination;
-- Role assignments are retained; reversing a schema does not delete users' roles.
