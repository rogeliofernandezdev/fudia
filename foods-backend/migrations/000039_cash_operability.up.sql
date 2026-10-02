ALTER TABLE cash_shifts ADD COLUMN business_date date;

UPDATE cash_shifts cs
SET business_date=(cs.opened_at AT TIME ZONE l.timezone)::date
FROM locations l
WHERE l.id=cs.location_id
  AND l.organization_id=cs.organization_id
  AND cs.business_date IS NULL;

ALTER TABLE cash_shifts ALTER COLUMN business_date SET NOT NULL;

ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements
  ADD COLUMN source_id uuid,
  ADD CONSTRAINT cash_movements_source_type_check
    CHECK (source_type IN (
      'manual',
      'cash_sale',
      'cash_refund',
      'cash_pull',
      'transfer_in',
      'transfer_out',
      'deposit',
      'adjustment'
    ));

CREATE INDEX cash_shifts_business_date_idx
  ON cash_shifts(organization_id, location_id, business_date DESC, opened_at DESC);

CREATE INDEX cash_movements_source_idx
  ON cash_movements(organization_id, location_id, source_type, source_id)
  WHERE source_id IS NOT NULL;
