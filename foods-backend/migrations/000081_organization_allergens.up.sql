-- Allergens: shared global seed (organization_id NULL) plus organization-specific entries.
-- normalize_allergen_name is the single definition of "same name": trimmed,
-- collapsed spaces, lowercase and without Spanish accents (ñ stays distinct).
CREATE FUNCTION normalize_allergen_name(value text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE
  AS $$ SELECT lower(translate(btrim(regexp_replace(value, '\s+', ' ', 'g')), 'ÁÉÍÓÚÜáéíóúü', 'AEIOUUaeiouu')) $$;

ALTER TABLE allergens DROP CONSTRAINT IF EXISTS allergens_name_key;
ALTER TABLE allergens
  ADD COLUMN organization_id uuid REFERENCES organizations(id),
  ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN normalized_name text GENERATED ALWAYS AS (normalize_allergen_name(name)) STORED;

CREATE UNIQUE INDEX allergens_global_name_uniq ON allergens(normalized_name) WHERE organization_id IS NULL;
CREATE UNIQUE INDEX allergens_org_name_uniq ON allergens(organization_id, normalized_name) WHERE organization_id IS NOT NULL;
