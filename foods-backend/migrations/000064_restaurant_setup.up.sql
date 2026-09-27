CREATE TABLE organization_operational_setup (
  organization_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
  service_mode text,
  completed_at timestamptz,
  completed_by uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (service_mode IS NULL OR service_mode IN ('counter','dine_in','mixed')),
  FOREIGN KEY (completed_by,organization_id) REFERENCES users(id,organization_id)
);

-- Las empresas existentes conservan su operación actual. Las que ya tienen
-- carta se consideran configuradas; el modo se infiere solo para esta
-- migración y puede modificarse posteriormente desde el asistente.
INSERT INTO organization_operational_setup(
  organization_id,service_mode,completed_at
)
SELECT
  o.id,
  CASE
    WHEN EXISTS(SELECT 1 FROM tables t WHERE t.organization_id=o.id AND t.active) THEN 'dine_in'
    ELSE 'counter'
  END,
  CASE
    WHEN EXISTS(SELECT 1 FROM products p WHERE p.organization_id=o.id AND p.active) THEN now()
    ELSE NULL
  END
FROM organizations o;
