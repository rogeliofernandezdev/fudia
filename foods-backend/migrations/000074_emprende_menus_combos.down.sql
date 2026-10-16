-- Do not overwrite module/plan edits performed after this migration.
UPDATE organization_modules m SET active=b.previous_active,updated_at=b.previous_updated_at
FROM migration_074_module_backup b
WHERE b.record_kind='organization' AND b.previous_active IS NOT NULL
  AND m.organization_id=b.target_id AND m.module_key='combos' AND m.updated_at=b.migrated_at;

DELETE FROM organization_modules m USING migration_074_module_backup b
WHERE b.record_kind='organization' AND b.previous_active IS NULL
  AND m.organization_id=b.target_id AND m.module_key='combos' AND m.updated_at=b.migrated_at;

UPDATE subscription_plans p SET module_keys=array_remove(p.module_keys,'combos'),updated_at=b.previous_updated_at
FROM migration_074_module_backup b
WHERE b.record_kind='plan' AND p.id=b.target_id AND p.updated_at=b.migrated_at;

DROP TABLE migration_074_module_backup;
