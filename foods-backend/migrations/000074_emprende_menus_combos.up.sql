-- Add the essential menu/combo capability without replacing customized plans.
-- Retain only changed values for a safe rollback of this migration.
CREATE TABLE migration_074_module_backup (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  record_kind text NOT NULL CHECK (record_kind IN ('plan','organization')),
  target_id uuid NOT NULL,
  previous_active boolean,
  previous_updated_at timestamptz,
  migrated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(record_kind,target_id)
);

INSERT INTO migration_074_module_backup(record_kind,target_id,previous_updated_at)
SELECT 'plan',id,updated_at FROM subscription_plans
WHERE code='emprende' AND NOT ('combos'=ANY(module_keys));

UPDATE subscription_plans SET module_keys=array_append(module_keys,'combos'),updated_at=now()
WHERE code='emprende' AND NOT ('combos'=ANY(module_keys));

INSERT INTO migration_074_module_backup(record_kind,target_id,previous_active,previous_updated_at)
SELECT 'organization',s.organization_id,m.active,m.updated_at
FROM organization_subscriptions s
JOIN subscription_plans p ON p.id=s.plan_id AND p.code='emprende'
LEFT JOIN organization_modules m ON m.organization_id=s.organization_id AND m.module_key='combos'
WHERE s.status<>'cancelled' AND (m.active IS NULL OR NOT m.active);

INSERT INTO organization_modules(organization_id,module_key,active)
SELECT target_id,'combos',true FROM migration_074_module_backup WHERE record_kind='organization'
ON CONFLICT(organization_id,module_key) DO UPDATE SET active=true,updated_at=now();
