-- Facturación electrónica becomes a ready module: Impulso and Escala include
-- it, Emprende does not. Current subscribers of those plans get the module
-- switched on; emitting stays off until each company enables its fiscal
-- profile. Pilots on other plans end, as with any ready module outside a plan.
CREATE TABLE migration_099_einvoicing_backup (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  record_kind text NOT NULL CHECK (record_kind IN ('plan','organization')),
  target_id uuid NOT NULL,
  previous_module_keys text[],
  previous_active boolean,
  previous_updated_at timestamptz,
  migrated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(record_kind,target_id)
);

INSERT INTO migration_099_einvoicing_backup(record_kind,target_id,previous_module_keys,previous_updated_at)
SELECT 'plan',id,module_keys,updated_at
FROM subscription_plans
WHERE code IN ('impulso','escala') AND NOT 'facturacion'=ANY(module_keys);

UPDATE subscription_plans p
SET module_keys=array_append(p.module_keys,'facturacion'),updated_at=now()
FROM migration_099_einvoicing_backup b
WHERE b.record_kind='plan' AND p.id=b.target_id;

INSERT INTO migration_099_einvoicing_backup(record_kind,target_id,previous_active,previous_updated_at)
SELECT 'organization',s.organization_id,m.active,m.updated_at
FROM organization_subscriptions s
JOIN subscription_plans p ON p.id=s.plan_id
LEFT JOIN organization_modules m ON m.organization_id=s.organization_id AND m.module_key='facturacion'
WHERE s.status<>'cancelled' AND 'facturacion'=ANY(p.module_keys) AND NOT COALESCE(m.active,false)
ON CONFLICT(record_kind,target_id) DO NOTHING;

INSERT INTO organization_modules(organization_id,module_key,active)
SELECT target_id,'facturacion',true
FROM migration_099_einvoicing_backup
WHERE record_kind='organization'
ON CONFLICT(organization_id,module_key)
DO UPDATE SET active=true,updated_at=now();
