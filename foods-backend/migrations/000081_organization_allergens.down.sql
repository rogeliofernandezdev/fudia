DELETE FROM allergens WHERE organization_id IS NOT NULL;
DROP INDEX IF EXISTS allergens_org_name_uniq;
DROP INDEX IF EXISTS allergens_global_name_uniq;
ALTER TABLE allergens DROP COLUMN normalized_name, DROP COLUMN created_at, DROP COLUMN organization_id;
ALTER TABLE allergens ADD CONSTRAINT allergens_name_key UNIQUE (name);
DROP FUNCTION IF EXISTS normalize_allergen_name(text);
