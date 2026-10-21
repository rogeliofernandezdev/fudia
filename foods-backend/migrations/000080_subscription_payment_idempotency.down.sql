DROP INDEX IF EXISTS subscription_payments_org_provider_external_reference_uidx;

-- Restore the original global provider/reference uniqueness when possible.
ALTER TABLE subscription_payments
  ADD CONSTRAINT subscription_payments_provider_external_reference_key
  UNIQUE(provider,external_reference);

DROP INDEX IF EXISTS subscription_payments_org_idempotency_key_uidx;
ALTER TABLE subscription_payments DROP COLUMN IF EXISTS idempotency_key;
