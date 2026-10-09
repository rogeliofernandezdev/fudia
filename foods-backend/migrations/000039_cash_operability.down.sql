DROP INDEX IF EXISTS cash_movements_source_idx;
DROP INDEX IF EXISTS cash_shifts_business_date_idx;

ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements DROP COLUMN IF EXISTS source_id;
ALTER TABLE cash_movements
  ADD CONSTRAINT cash_movements_source_type_check CHECK (source_type IN ('manual'));

ALTER TABLE cash_shifts DROP COLUMN IF EXISTS business_date;
