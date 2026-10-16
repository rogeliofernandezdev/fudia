DROP INDEX IF EXISTS cash_shifts_register_idx;
DROP INDEX IF EXISTS cash_shifts_open_register_uq;
ALTER TABLE cash_shifts DROP CONSTRAINT IF EXISTS cash_shifts_register_scope_fk;
ALTER TABLE cash_shifts DROP COLUMN IF EXISTS cash_register_id;
DROP TABLE IF EXISTS cash_registers;
