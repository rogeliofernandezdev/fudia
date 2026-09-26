UPDATE roles
SET permissions=array_remove(permissions,'cash.expected.read')
WHERE system_key IN ('location_manager','shift_supervisor','accounting','auditor');

DROP INDEX IF EXISTS cash_movements_source_unique_idx;
DROP TABLE IF EXISTS payment_refunds;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS cash_operations;
DROP TABLE IF EXISTS cash_count_lines;
DROP TABLE IF EXISTS cash_shift_users;
ALTER TABLE cash_registers DROP COLUMN IF EXISTS blind_close;
