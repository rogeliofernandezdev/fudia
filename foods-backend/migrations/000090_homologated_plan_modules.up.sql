-- Cumulative commercial plans built only from ready modules:
-- Emprende sells and serves one location; Impulso adds supply, recipes,
-- reservations and expenses; Escala keeps Impulso modules with higher limits.
-- Current Emprende subscribers keep Inventario and Compras on a non-sellable plan.
-- Prices, limits and role assignments are not changed.
CREATE TABLE migration_090_plan_backup (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  record_kind text NOT NULL CHECK (record_kind IN ('plan','created_plan','subscription','organization')),
  target_id uuid NOT NULL,
  module_key text NOT NULL DEFAULT '',
  previous_module_keys text[],
  previous_plan_id uuid,
  previous_active boolean,
  previous_updated_at timestamptz,
  migrated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(record_kind,target_id,module_key)
);

-- Grandfathered plan: previous Emprende modules plus Concierge.
INSERT INTO subscription_plans(
  code,name,description,currency,monthly_price,annual_price,trial_days,
  max_locations,max_users,module_keys,terms_version,active
)
SELECT 'emprende-abastecimiento','Emprende con abastecimiento',
       'Plan anterior de Emprende que conserva Inventario y Compras para las empresas suscritas antes de la homologación de planes.',
       p.currency,p.monthly_price,p.annual_price,p.trial_days,p.max_locations,p.max_users,
       CASE WHEN 'whatsapp_bot'=ANY(p.module_keys) THEN p.module_keys ELSE array_append(p.module_keys,'whatsapp_bot') END,
       p.terms_version,false
FROM subscription_plans p
WHERE p.code='emprende'
  AND p.module_keys && ARRAY['inventario','compras']::text[]
  AND EXISTS (
    SELECT 1
    FROM organization_subscriptions s
    JOIN organization_modules m
      ON m.organization_id=s.organization_id AND m.module_key IN ('inventario','compras') AND m.active
    WHERE s.plan_id=p.id AND s.status<>'cancelled'
  )
ON CONFLICT(code) DO NOTHING;

INSERT INTO migration_090_plan_backup(record_kind,target_id)
SELECT 'created_plan',id FROM subscription_plans
WHERE code='emprende-abastecimiento' AND updated_at=now()
ON CONFLICT(record_kind,target_id,module_key) DO NOTHING;

INSERT INTO migration_090_plan_backup(record_kind,target_id,previous_plan_id,previous_updated_at)
SELECT 'subscription',s.id,s.plan_id,s.updated_at
FROM organization_subscriptions s
JOIN subscription_plans p ON p.id=s.plan_id AND p.code='emprende'
WHERE s.status<>'cancelled'
  AND EXISTS (SELECT 1 FROM subscription_plans WHERE code='emprende-abastecimiento')
  AND EXISTS (
    SELECT 1 FROM organization_modules m
    WHERE m.organization_id=s.organization_id AND m.module_key IN ('inventario','compras') AND m.active
  )
ON CONFLICT(record_kind,target_id,module_key) DO NOTHING;

UPDATE organization_subscriptions s
SET plan_id=g.id,updated_at=now()
FROM migration_090_plan_backup b,subscription_plans g
WHERE b.record_kind='subscription' AND s.id=b.target_id AND g.code='emprende-abastecimiento';

-- Official module sets.
WITH target(code,module_keys) AS (VALUES
  ('emprende',ARRAY['reportes','pos','pedidos','cocina','mesas','caja','productos','combos','clientes','locales','fiscal','usuarios','whatsapp_bot']::text[]),
  ('impulso',ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','compras','clientes','locales','costos','fiscal','usuarios','whatsapp_bot']::text[]),
  ('escala',ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','compras','clientes','locales','costos','fiscal','usuarios','whatsapp_bot']::text[])
)
INSERT INTO migration_090_plan_backup(record_kind,target_id,previous_module_keys,previous_updated_at)
SELECT 'plan',p.id,p.module_keys,p.updated_at
FROM subscription_plans p JOIN target t ON t.code=p.code
WHERE p.module_keys IS DISTINCT FROM t.module_keys
ON CONFLICT(record_kind,target_id,module_key) DO NOTHING;

WITH target(code,module_keys) AS (VALUES
  ('emprende',ARRAY['reportes','pos','pedidos','cocina','mesas','caja','productos','combos','clientes','locales','fiscal','usuarios','whatsapp_bot']::text[]),
  ('impulso',ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','compras','clientes','locales','costos','fiscal','usuarios','whatsapp_bot']::text[]),
  ('escala',ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','compras','clientes','locales','costos','fiscal','usuarios','whatsapp_bot']::text[])
)
UPDATE subscription_plans p
SET module_keys=t.module_keys,updated_at=now()
FROM target t
WHERE t.code=p.code AND p.module_keys IS DISTINCT FROM t.module_keys;

-- Align the modules whose plan membership changed for current subscribers.
INSERT INTO migration_090_plan_backup(record_kind,target_id,module_key,previous_active,previous_updated_at)
SELECT 'organization',s.organization_id,changed.key,m.active,m.updated_at
FROM organization_subscriptions s
JOIN subscription_plans p
  ON p.id=s.plan_id AND p.code IN ('emprende','impulso','escala','emprende-abastecimiento')
CROSS JOIN unnest(ARRAY['inventario','compras','costos','whatsapp_bot']::text[]) AS changed(key)
LEFT JOIN organization_modules m ON m.organization_id=s.organization_id AND m.module_key=changed.key
WHERE s.status<>'cancelled' AND COALESCE(m.active,false)<>(changed.key=ANY(p.module_keys))
ON CONFLICT(record_kind,target_id,module_key) DO NOTHING;

INSERT INTO organization_modules(organization_id,module_key,active)
SELECT b.target_id,b.module_key,b.module_key=ANY(p.module_keys)
FROM migration_090_plan_backup b
JOIN organization_subscriptions s ON s.organization_id=b.target_id
JOIN subscription_plans p ON p.id=s.plan_id
WHERE b.record_kind='organization'
ON CONFLICT(organization_id,module_key)
DO UPDATE SET active=EXCLUDED.active,updated_at=now();
