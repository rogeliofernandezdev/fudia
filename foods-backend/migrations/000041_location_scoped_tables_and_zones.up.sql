-- Scope restaurant floor entities by location without losing historical orders.
ALTER TABLE tables ADD COLUMN IF NOT EXISTS location_id uuid;
ALTER TABLE zones ADD COLUMN IF NOT EXISTS location_id uuid;

ALTER TABLE tables DROP CONSTRAINT IF EXISTS tables_organization_id_name_key;
ALTER TABLE zones DROP CONSTRAINT IF EXISTS zones_organization_id_name_key;

-- Keep the original table row in the most recently used location.
UPDATE tables t
SET location_id=COALESCE(
  (
    SELECT o.location_id
    FROM orders o
    WHERE o.organization_id=t.organization_id AND o.table_id=t.id
    ORDER BY o.created_at DESC,o.id
    LIMIT 1
  ),
  (
    SELECT l.id
    FROM locations l
    WHERE l.organization_id=t.organization_id AND l.active
    ORDER BY l.created_at,l.id
    LIMIT 1
  ),
  (
    SELECT l.id
    FROM locations l
    WHERE l.organization_id=t.organization_id
    ORDER BY l.created_at,l.id
    LIMIT 1
  )
)
WHERE t.location_id IS NULL;

-- If a legacy table was used by more than one location, create one physical
-- table row per additional location. New QR tokens keep public links unique.
INSERT INTO tables(
  organization_id,location_id,name,seats,zone,active,created_at,updated_at,qr_token,qr_enabled
)
SELECT DISTINCT
  t.organization_id,o.location_id,t.name,t.seats,t.zone,t.active,t.created_at,t.updated_at,
  encode(gen_random_bytes(16),'hex'),t.qr_enabled
FROM tables t
JOIN orders o
  ON o.organization_id=t.organization_id AND o.table_id=t.id
WHERE t.location_id IS DISTINCT FROM o.location_id
  AND NOT EXISTS (
    SELECT 1
    FROM tables existing
    WHERE existing.organization_id=t.organization_id
      AND existing.location_id=o.location_id
      AND existing.name=t.name
  );

-- Repoint historical orders to the location-specific copy.
UPDATE orders o
SET table_id=target.id
FROM tables original
JOIN tables target
  ON target.organization_id=original.organization_id
 AND target.name=original.name
WHERE o.table_id=original.id
  AND o.organization_id=original.organization_id
  AND target.location_id=o.location_id
  AND original.location_id IS DISTINCT FROM o.location_id;

-- Zones are configuration; replicate the existing configuration to every
-- current location so no site loses its floor filters after the migration.
UPDATE zones z
SET location_id=COALESCE(
  (
    SELECT l.id
    FROM locations l
    WHERE l.organization_id=z.organization_id AND l.active
    ORDER BY l.created_at,l.id
    LIMIT 1
  ),
  (
    SELECT l.id
    FROM locations l
    WHERE l.organization_id=z.organization_id
    ORDER BY l.created_at,l.id
    LIMIT 1
  )
)
WHERE z.location_id IS NULL;

INSERT INTO zones(organization_id,location_id,name,sort_order,active,created_at)
SELECT z.organization_id,l.id,z.name,z.sort_order,z.active,z.created_at
FROM zones z
JOIN locations l ON l.organization_id=z.organization_id
WHERE z.location_id IS DISTINCT FROM l.id
  AND NOT EXISTS (
    SELECT 1
    FROM zones existing
    WHERE existing.organization_id=z.organization_id
      AND existing.location_id=l.id
      AND existing.name=z.name
  );

ALTER TABLE tables ALTER COLUMN location_id SET NOT NULL;
ALTER TABLE zones ALTER COLUMN location_id SET NOT NULL;

ALTER TABLE tables
  ADD CONSTRAINT tables_location_org_fkey
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id);

ALTER TABLE zones
  ADD CONSTRAINT zones_location_org_fkey
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id);

ALTER TABLE tables
  ADD CONSTRAINT tables_organization_location_name_key UNIQUE(organization_id,location_id,name),
  ADD CONSTRAINT tables_id_org_location_key UNIQUE(id,organization_id,location_id);

ALTER TABLE zones
  ADD CONSTRAINT zones_organization_location_name_key UNIQUE(organization_id,location_id,name);

DROP INDEX IF EXISTS tables_scope_idx;
CREATE INDEX tables_scope_idx ON tables(organization_id,location_id,active,name);
DROP INDEX IF EXISTS zones_scope_idx;
CREATE INDEX zones_scope_idx ON zones(organization_id,location_id,active,sort_order,name);

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_table_id_organization_id_fkey;
ALTER TABLE orders
  ADD CONSTRAINT orders_table_location_fkey
  FOREIGN KEY(table_id,organization_id,location_id)
  REFERENCES tables(id,organization_id,location_id);
