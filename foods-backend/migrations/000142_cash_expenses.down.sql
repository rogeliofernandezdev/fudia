-- Conserva los movimientos de caja: vuelven a ser egresos e ingresos manuales.
UPDATE cash_movements SET source_type='manual',source_id=NULL WHERE source_type IN ('expense','expense_void');
ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements ADD CONSTRAINT cash_movements_source_type_check
 CHECK(source_type IN ('manual','cash_sale','cash_refund','cash_pull','deposit','transfer_in','transfer_out','adjustment','cash_tip','tip_payout'));
DROP INDEX IF EXISTS expenses_cash_shift_idx;
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_cash_shift_fk;
ALTER TABLE expenses DROP COLUMN IF EXISTS cash_shift_id;
