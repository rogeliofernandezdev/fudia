DROP INDEX IF EXISTS orders_one_open_table_uq;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_table_location_fkey;

-- Collapse location-specific table copies back to one organization-wide row.
WITH canonical AS (
  SELECT DISTINCT ON (organization_id,name)
    organization_id,name,id
  FROM tables
  ORDER BY organization_id,name,created_at,id
)
UPDATE orders o
SET table_id=canonical.id
FROM tables current_table
JOIN canonical
  ON canonical.organization_id=current_table.organization_id
 AND canonical.name=current_table.name
WHERE o.table_id=current_table.id
  AND o.organization_id=current_table.organization_id
  AND o.table_id<>canonical.id;

WITH canonical AS (
  SELECT DISTINCT ON (organization_id,name)
    organization_id,name,id
  FROM tables
  ORDER BY organization_id,name,created_at,id
)
DELETE FROM tables t
USING canonical
WHERE t.organization_id=canonical.organization_id
  AND t.name=canonical.name
  AND t.id<>canonical.id;

WITH canonical AS (
  SELECT DISTINCT ON (organization_id,name)
    organization_id,name,id
  FROM zones
  ORDER BY organization_id,name,created_at,id
)
DELETE FROM zones z
USING canonical
WHERE z.organization_id=canonical.organization_id
  AND z.name=canonical.name
  AND z.id<>canonical.id;

ALTER TABLE tables DROP CONSTRAINT IF EXISTS tables_location_org_fkey;
ALTER TABLE zones DROP CONSTRAINT IF EXISTS zones_location_org_fkey;
ALTER TABLE tables DROP CONSTRAINT IF EXISTS tables_organization_location_name_key;
ALTER TABLE tables DROP CONSTRAINT IF EXISTS tables_id_org_location_key;
ALTER TABLE zones DROP CONSTRAINT IF EXISTS zones_organization_location_name_key;

DROP INDEX IF EXISTS tables_scope_idx;
DROP INDEX IF EXISTS zones_scope_idx;

ALTER TABLE tables DROP COLUMN IF EXISTS location_id;
ALTER TABLE zones DROP COLUMN IF EXISTS location_id;

ALTER TABLE tables ADD CONSTRAINT tables_organization_id_name_key UNIQUE(organization_id,name);
ALTER TABLE zones ADD CONSTRAINT zones_organization_id_name_key UNIQUE(organization_id,name);
CREATE INDEX tables_scope_idx ON tables(organization_id,active,name);
CREATE INDEX zones_scope_idx ON zones(organization_id,active,sort_order,name);

ALTER TABLE orders
  ADD CONSTRAINT orders_table_id_organization_id_fkey
  FOREIGN KEY(table_id,organization_id) REFERENCES tables(id,organization_id);
