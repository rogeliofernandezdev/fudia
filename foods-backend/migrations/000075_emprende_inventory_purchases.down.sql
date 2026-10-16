-- Preserve edits made after migration; restore only the rows changed here.
UPDATE organization_modules m SET active=b.previous_active,updated_at=b.previous_updated_at
FROM migration_075_module_backup b
WHERE b.record_kind='organization' AND b.previous_active IS NOT NULL
  AND m.organization_id=b.target_id AND m.module_key=b.module_key AND m.updated_at=b.migrated_at;

DELETE FROM organization_modules m USING migration_075_module_backup b
WHERE b.record_kind='organization' AND b.previous_active IS NULL
  AND m.organization_id=b.target_id AND m.module_key=b.module_key AND m.updated_at=b.migrated_at;

UPDATE subscription_plans p SET module_keys=b.previous_module_keys,updated_at=b.previous_updated_at
FROM migration_075_module_backup b
WHERE b.record_kind='plan' AND p.id=b.target_id AND p.updated_at=b.migrated_at;

DROP TABLE migration_075_module_backup;
