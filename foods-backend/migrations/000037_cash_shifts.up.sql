CREATE TABLE cash_shifts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  code text NOT NULL DEFAULT ('CAJ-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 8))),
  status text NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed')),
  opening_amount numeric(12,2) NOT NULL DEFAULT 0 CHECK (opening_amount >= 0),
  closing_expected_amount numeric(12,2),
  closing_counted_amount numeric(12,2),
  variance_amount numeric(12,2),
  opening_note text NOT NULL DEFAULT '',
  closing_note text NOT NULL DEFAULT '',
  opened_by uuid NOT NULL,
  closed_by uuid,
  opened_at timestamptz NOT NULL DEFAULT now(),
  closed_at timestamptz,
  UNIQUE (organization_id, code),
  UNIQUE (id, organization_id, location_id),
  FOREIGN KEY (location_id, organization_id) REFERENCES locations(id, organization_id),
  FOREIGN KEY (opened_by, organization_id) REFERENCES users(id, organization_id),
  FOREIGN KEY (closed_by, organization_id) REFERENCES users(id, organization_id),
  CHECK (
    (status='open' AND closed_at IS NULL AND closing_expected_amount IS NULL AND closing_counted_amount IS NULL AND variance_amount IS NULL)
    OR
    (status='closed' AND closed_at IS NOT NULL AND closing_expected_amount IS NOT NULL AND closing_counted_amount IS NOT NULL AND variance_amount IS NOT NULL)
  )
);

CREATE UNIQUE INDEX cash_shifts_open_user_location_uq
  ON cash_shifts(organization_id, location_id, opened_by)
  WHERE status='open';

CREATE INDEX cash_shifts_scope_idx
  ON cash_shifts(organization_id, location_id, opened_at DESC);

CREATE TABLE cash_movements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  shift_id uuid NOT NULL,
  movement_type text NOT NULL CHECK (movement_type IN ('income','expense')),
  source_type text NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual')),
  amount numeric(12,2) NOT NULL CHECK (amount > 0),
  reason text NOT NULL,
  note text NOT NULL DEFAULT '',
  created_by uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (shift_id, organization_id, location_id) REFERENCES cash_shifts(id, organization_id, location_id),
  FOREIGN KEY (location_id, organization_id) REFERENCES locations(id, organization_id),
  FOREIGN KEY (created_by, organization_id) REFERENCES users(id, organization_id)
);

CREATE INDEX cash_movements_shift_idx
  ON cash_movements(organization_id, location_id, shift_id, created_at DESC);
