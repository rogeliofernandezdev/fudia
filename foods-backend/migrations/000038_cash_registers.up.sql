CREATE TABLE cash_registers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  code text NOT NULL DEFAULT ('CJ-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 6))),
  name text NOT NULL,
  active boolean NOT NULL DEFAULT true,
  created_by uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, location_id, code),
  UNIQUE (id, organization_id, location_id),
  FOREIGN KEY (location_id, organization_id) REFERENCES locations(id, organization_id),
  FOREIGN KEY (created_by, organization_id) REFERENCES users(id, organization_id),
  CHECK (length(btrim(name)) BETWEEN 1 AND 80)
);

CREATE UNIQUE INDEX cash_registers_name_uq
  ON cash_registers(organization_id, location_id, lower(btrim(name)));

CREATE INDEX cash_registers_scope_idx
  ON cash_registers(organization_id, location_id, active, name);

ALTER TABLE cash_shifts ADD COLUMN cash_register_id uuid;

INSERT INTO cash_registers(organization_id,location_id,name,created_by)
SELECT DISTINCT ON (cs.organization_id,cs.location_id)
       cs.organization_id,cs.location_id,'Caja principal',cs.opened_by
FROM cash_shifts cs
ORDER BY cs.organization_id,cs.location_id,cs.opened_at;

UPDATE cash_shifts cs
SET cash_register_id=cr.id
FROM cash_registers cr
WHERE cr.organization_id=cs.organization_id
  AND cr.location_id=cs.location_id
  AND cr.name='Caja principal'
  AND cs.cash_register_id IS NULL;

ALTER TABLE cash_shifts
  ALTER COLUMN cash_register_id SET NOT NULL,
  ADD CONSTRAINT cash_shifts_register_scope_fk
    FOREIGN KEY (cash_register_id,organization_id,location_id)
    REFERENCES cash_registers(id,organization_id,location_id);

CREATE UNIQUE INDEX cash_shifts_open_register_uq
  ON cash_shifts(organization_id, location_id, cash_register_id)
  WHERE status='open';

CREATE INDEX cash_shifts_register_idx
  ON cash_shifts(organization_id, location_id, cash_register_id, opened_at DESC);
