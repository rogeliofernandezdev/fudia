DROP INDEX IF EXISTS inventory_entries_presentation_idx;

ALTER TABLE inventory_entries
  DROP CONSTRAINT IF EXISTS inventory_entries_presentation_factor_check,
  DROP CONSTRAINT IF EXISTS inventory_entries_presentation_type_check,
  DROP CONSTRAINT IF EXISTS inventory_entries_presentation_scope_fk,
  DROP COLUMN IF EXISTS stock_quantity,
  DROP COLUMN IF EXISTS units_per_presentation,
  DROP COLUMN IF EXISTS presentation_type,
  DROP COLUMN IF EXISTS presentation_id;

DROP TABLE IF EXISTS inventory_presentations;
