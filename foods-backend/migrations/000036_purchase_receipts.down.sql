DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM purchase_receipts LIMIT 1) THEN
    RAISE EXCEPTION 'cannot rollback purchase receipts while receipt data exists';
  END IF;
END $$;

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_source_type_check,
  ADD CONSTRAINT stock_movements_source_type_check
    CHECK (source_type IN ('inventory_entry','order','inventory_adjustment'));

DROP INDEX IF EXISTS purchase_receipts_order_idx;
DROP TABLE IF EXISTS purchase_receipt_items;
DROP TABLE IF EXISTS purchase_receipts;

ALTER TABLE purchase_order_items
  DROP COLUMN IF EXISTS received_quantity;

ALTER TABLE purchase_orders
  DROP CONSTRAINT IF EXISTS purchase_orders_status_check,
  ADD CONSTRAINT purchase_orders_status_check
    CHECK(status IN ('draft','pending_approval','approved','received','cancelled'));
