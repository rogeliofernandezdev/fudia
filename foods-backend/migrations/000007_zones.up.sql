CREATE TABLE zones (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  name text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(organization_id,name),
  UNIQUE(id,organization_id)
);
CREATE INDEX zones_scope_idx ON zones(organization_id,active,sort_order,name);
