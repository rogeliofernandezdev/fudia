DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM inventory_adjustments LIMIT 1) THEN
    RAISE EXCEPTION 'cannot rollback inventory adjustments while adjustment data exists';
  END IF;
END $$;

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_source_type_check,
  ADD CONSTRAINT stock_movements_source_type_check
    CHECK (source_type IN ('inventory_entry','order'));

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check,
  ADD CONSTRAINT stock_movements_movement_type_check
    CHECK (movement_type IN ('entry','sale','sale_reversal','sale_adjustment'));

DROP INDEX IF EXISTS inventory_adjustments_scope_idx;
DROP TABLE IF EXISTS inventory_adjustments;
