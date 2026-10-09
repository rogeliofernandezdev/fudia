CREATE TABLE cash_shift_reports (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  shift_id uuid NOT NULL,
  report jsonb NOT NULL CHECK (jsonb_typeof(report) = 'object'),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, location_id, shift_id),
  FOREIGN KEY (shift_id, organization_id, location_id)
    REFERENCES cash_shifts(id, organization_id, location_id)
);
