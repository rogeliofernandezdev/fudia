ALTER TABLE tables
  DROP CONSTRAINT IF EXISTS tables_qr_token_unique;
DROP INDEX IF EXISTS tables_qr_token_idx;
ALTER TABLE tables
  DROP COLUMN IF EXISTS qr_token,
  DROP COLUMN IF EXISTS qr_enabled;
