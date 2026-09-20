ALTER TABLE organizations ALTER COLUMN tax_id TYPE varchar(32);

ALTER TABLE locations
  ADD COLUMN timezone text NOT NULL DEFAULT 'America/Lima',
  ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

UPDATE locations l
SET timezone=o.timezone
FROM organizations o
WHERE o.id=l.organization_id;
