DROP INDEX IF EXISTS inventory_entries_code_uq;
ALTER TABLE inventory_entries DROP COLUMN IF EXISTS code;
