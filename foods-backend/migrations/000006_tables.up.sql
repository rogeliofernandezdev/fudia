CREATE TABLE tables (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  name text NOT NULL,
  seats integer NOT NULL DEFAULT 2 CHECK(seats>=1),
  zone text NOT NULL DEFAULT '',
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(organization_id,name),
  UNIQUE(id,organization_id)
);
CREATE INDEX tables_scope_idx ON tables(organization_id,active,name);
