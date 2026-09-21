DROP TABLE IF EXISTS inventory_transfer_items;
DROP TABLE IF EXISTS inventory_transfers;
DROP TABLE IF EXISTS purchase_return_items;
DROP TABLE IF EXISTS purchase_returns;
DROP INDEX IF EXISTS purchase_receipts_idempotency_uq;
ALTER TABLE purchase_receipt_items DROP COLUMN IF EXISTS unit_cost;
ALTER TABLE purchase_receipts DROP COLUMN IF EXISTS idempotency_key;
ALTER TABLE stock_movements
  DROP COLUMN IF EXISTS balance_value_after,
  DROP COLUMN IF EXISTS value_delta,
  DROP COLUMN IF EXISTS unit_cost;
ALTER TABLE stock_balances DROP COLUMN IF EXISTS average_unit_cost;
DROP TABLE IF EXISTS inventory_location_settings;
DROP INDEX IF EXISTS suppliers_org_tax_id_uq;

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check,
  ADD CONSTRAINT stock_movements_movement_type_check
    CHECK(movement_type IN ('entry','sale','sale_reversal','sale_adjustment','inventory_adjustment'));
ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_source_type_check,
  ADD CONSTRAINT stock_movements_source_type_check
    CHECK(source_type IN ('inventory_entry','order','inventory_adjustment','purchase_receipt'));
