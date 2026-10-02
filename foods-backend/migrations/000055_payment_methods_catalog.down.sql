ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_payment_method_fk;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_payment_method_fk;

UPDATE expenses SET payment_method='bank_transfer' WHERE payment_method='transfer';
UPDATE expenses SET payment_method='digital_wallet' WHERE payment_method='wallet';
UPDATE payments SET method='transfer' WHERE method='wallet';

ALTER TABLE payments
  ADD CONSTRAINT payments_method_check CHECK (method IN ('cash','card','transfer','other'));

ALTER TABLE expenses
  ADD CONSTRAINT expenses_payment_method_check
  CHECK (payment_method IN ('cash','bank_transfer','card','digital_wallet','other'));

DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS payment_method_templates;
