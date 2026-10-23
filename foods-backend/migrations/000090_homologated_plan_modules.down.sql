-- Preserve edits made after migration; restore only the rows changed here.
UPDATE organization_modules m SET active=b.previous_active,updated_at=b.previous_updated_at
FROM migration_090_plan_backup b
WHERE b.record_kind='organization' AND b.previous_active IS NOT NULL
  AND m.organization_id=b.target_id AND m.module_key=b.module_key AND m.updated_at=b.migrated_at;

DELETE FROM organization_modules m USING migration_090_plan_backup b
WHERE b.record_kind='organization' AND b.previous_active IS NULL
  AND m.organization_id=b.target_id AND m.module_key=b.module_key AND m.updated_at=b.migrated_at;

UPDATE subscription_plans p SET module_keys=b.previous_module_keys,updated_at=b.previous_updated_at
FROM migration_090_plan_backup b
WHERE b.record_kind='plan' AND p.id=b.target_id AND p.updated_at=b.migrated_at;

UPDATE organization_subscriptions s SET plan_id=b.previous_plan_id,updated_at=b.previous_updated_at
FROM migration_090_plan_backup b
WHERE b.record_kind='subscription' AND s.id=b.target_id AND s.updated_at=b.migrated_at;

DELETE FROM subscription_plans p USING migration_090_plan_backup b
WHERE b.record_kind='created_plan' AND p.id=b.target_id
  AND NOT EXISTS (SELECT 1 FROM organization_subscriptions s WHERE s.plan_id=p.id);

DROP TABLE migration_090_plan_backup;
