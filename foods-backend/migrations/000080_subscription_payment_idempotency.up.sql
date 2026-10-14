ALTER TABLE subscription_payments ADD COLUMN idempotency_key text;

CREATE UNIQUE INDEX subscription_payments_org_idempotency_key_uidx
  ON subscription_payments(organization_id,idempotency_key)
  WHERE idempotency_key IS NOT NULL;

ALTER TABLE subscription_payments
  DROP CONSTRAINT IF EXISTS subscription_payments_provider_external_reference_key;
CREATE UNIQUE INDEX subscription_payments_org_provider_external_reference_uidx
  ON subscription_payments(organization_id,provider,external_reference)
  WHERE external_reference IS NOT NULL;
