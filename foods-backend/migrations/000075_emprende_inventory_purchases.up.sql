-- Extend the existing plan only; retain prices, limits and role assignments.
CREATE TABLE IF NOT EXISTS migration_075_module_backup (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  record_kind text NOT NULL CHECK (record_kind IN ('plan','organization')),
  target_id uuid NOT NULL,
  module_key text NOT NULL DEFAULT '',
  previous_module_keys text[],
  previous_active boolean,
  previous_updated_at timestamptz,
  migrated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(record_kind,target_id,module_key)
);

INSERT INTO migration_075_module_backup(record_kind,target_id,previous_module_keys,previous_updated_at)
SELECT 'plan',id,module_keys,updated_at FROM subscription_plans
WHERE code='emprende' AND NOT (module_keys @> ARRAY['inventario','compras']::text[])
ON CONFLICT(record_kind,target_id,module_key) DO NOTHING;

UPDATE subscription_plans p
SET module_keys=p.module_keys || ARRAY(
      SELECT key FROM unnest(ARRAY['inventario','compras']::text[]) AS requested(key)
      WHERE NOT (key=ANY(p.module_keys))
    ),updated_at=now()
WHERE p.code='emprende' AND NOT (p.module_keys @> ARRAY['inventario','compras']::text[]);

INSERT INTO migration_075_module_backup(record_kind,target_id,module_key,previous_active,previous_updated_at)
SELECT 'organization',s.organization_id,requested.key,m.active,m.updated_at
FROM organization_subscriptions s
JOIN subscription_plans p ON p.id=s.plan_id AND p.code='emprende'
CROSS JOIN unnest(ARRAY['inventario','compras']::text[]) AS requested(key)
LEFT JOIN organization_modules m ON m.organization_id=s.organization_id AND m.module_key=requested.key
WHERE s.status<>'cancelled' AND (m.active IS NULL OR NOT m.active)
ON CONFLICT(record_kind,target_id,module_key) DO NOTHING;

INSERT INTO organization_modules(organization_id,module_key,active)
SELECT b.target_id,b.module_key,true
FROM migration_075_module_backup b
JOIN organization_subscriptions s ON s.organization_id=b.target_id AND s.status<>'cancelled'
JOIN subscription_plans p ON p.id=s.plan_id AND p.code='emprende'
WHERE b.record_kind='organization'
ON CONFLICT(organization_id,module_key)
DO UPDATE SET active=true,updated_at=now() WHERE NOT organization_modules.active;
