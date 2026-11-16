ALTER TABLE pay_pairing_codes
  DROP COLUMN IF EXISTS scanned_model,
  DROP COLUMN IF EXISTS scanned_at;
