ALTER TABLE organization_subscriptions
  ADD COLUMN billing_anchor_day smallint
  CHECK (billing_anchor_day IS NULL OR billing_anchor_day BETWEEN 1 AND 31);
