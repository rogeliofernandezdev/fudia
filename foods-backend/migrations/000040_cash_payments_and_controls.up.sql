ALTER TABLE cash_registers
  ADD COLUMN blind_close boolean NOT NULL DEFAULT false;

CREATE TABLE cash_shift_users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  shift_id uuid NOT NULL,
  user_id uuid NOT NULL,
  assigned_by uuid NOT NULL,
  assigned_at timestamptz NOT NULL DEFAULT now(),
  unassigned_at timestamptz,
  FOREIGN KEY (shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
  FOREIGN KEY (user_id,organization_id) REFERENCES users(id,organization_id),
  FOREIGN KEY (assigned_by,organization_id) REFERENCES users(id,organization_id)
);

CREATE UNIQUE INDEX cash_shift_users_active_shift_user_uq
  ON cash_shift_users(organization_id,location_id,shift_id,user_id)
  WHERE unassigned_at IS NULL;

CREATE UNIQUE INDEX cash_shift_users_active_user_uq
  ON cash_shift_users(organization_id,location_id,user_id)
  WHERE unassigned_at IS NULL;

INSERT INTO cash_shift_users(organization_id,location_id,shift_id,user_id,assigned_by,assigned_at)
SELECT organization_id,location_id,id,opened_by,opened_by,opened_at
FROM cash_shifts
WHERE status='open'
ON CONFLICT DO NOTHING;

CREATE TABLE cash_count_lines (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  shift_id uuid NOT NULL,
  denomination numeric(12,2) NOT NULL CHECK (denomination > 0),
  quantity integer NOT NULL CHECK (quantity >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (shift_id,denomination),
  FOREIGN KEY (shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id)
);

CREATE TABLE cash_operations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  operation_type text NOT NULL CHECK (operation_type IN ('cash_pull','deposit','transfer')),
  source_shift_id uuid NOT NULL,
  target_shift_id uuid,
  amount numeric(12,2) NOT NULL CHECK (amount > 0),
  reason text NOT NULL,
  note text NOT NULL DEFAULT '',
  created_by uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (source_shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
  FOREIGN KEY (target_shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
  FOREIGN KEY (created_by,organization_id) REFERENCES users(id,organization_id),
  CHECK (
    (operation_type='transfer' AND target_shift_id IS NOT NULL AND target_shift_id<>source_shift_id)
    OR
    (operation_type IN ('cash_pull','deposit') AND target_shift_id IS NULL)
  )
);

CREATE INDEX cash_operations_source_idx
  ON cash_operations(organization_id,location_id,source_shift_id,created_at DESC);

CREATE TABLE payments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  order_id uuid NOT NULL,
  shift_id uuid NOT NULL,
  method text NOT NULL CHECK (method IN ('cash','card','transfer','other')),
  amount numeric(12,2) NOT NULL CHECK (amount > 0),
  reference text NOT NULL DEFAULT '',
  created_by uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id,location_id),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id),
  FOREIGN KEY (shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
  FOREIGN KEY (created_by,organization_id) REFERENCES users(id,organization_id)
);

CREATE INDEX payments_order_idx
  ON payments(organization_id,location_id,order_id,created_at DESC);

CREATE INDEX payments_shift_idx
  ON payments(organization_id,location_id,shift_id,created_at DESC);

CREATE TABLE payment_refunds (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  payment_id uuid NOT NULL,
  shift_id uuid NOT NULL,
  amount numeric(12,2) NOT NULL CHECK (amount > 0),
  reason text NOT NULL,
  note text NOT NULL DEFAULT '',
  created_by uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id,location_id),
  FOREIGN KEY (payment_id,organization_id,location_id) REFERENCES payments(id,organization_id,location_id),
  FOREIGN KEY (shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
  FOREIGN KEY (created_by,organization_id) REFERENCES users(id,organization_id)
);

CREATE INDEX payment_refunds_payment_idx
  ON payment_refunds(organization_id,location_id,payment_id,created_at DESC);

CREATE UNIQUE INDEX cash_movements_source_unique_idx
  ON cash_movements(organization_id,location_id,source_type,source_id)
  WHERE source_id IS NOT NULL;

UPDATE roles
SET permissions = CASE
  WHEN system_key IN ('location_manager','shift_supervisor','accounting','auditor')
       AND NOT permissions @> ARRAY['cash.expected.read']::text[]
    THEN array_append(permissions,'cash.expected.read')
  ELSE permissions
END
WHERE system_key IN ('location_manager','shift_supervisor','accounting','auditor');
