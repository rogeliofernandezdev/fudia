-- Purchase packaging is a tenant catalog, not a closed frontend enum.
CREATE TABLE inventory_presentation_type_templates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  name text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0
);
INSERT INTO inventory_presentation_type_templates(code,name,sort_order) VALUES
 ('unit','Unidad base',0),('package','Paquete',10),('box','Caja',20),
 ('bag','Bolsa',30),('sack','Saco',40),('drum','Bidón',50);
CREATE TABLE inventory_presentation_types (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  code text NOT NULL,
  name text NOT NULL,
  sort_order integer NOT NULL DEFAULT 1000,
  UNIQUE(organization_id,code)
);
CREATE TABLE inventory_unit_combination_templates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  unit text NOT NULL,
  presentation_type text NOT NULL REFERENCES inventory_presentation_type_templates(code),
  is_default boolean NOT NULL DEFAULT false,
  UNIQUE(unit,presentation_type)
);
INSERT INTO inventory_unit_combination_templates(unit,presentation_type,is_default) VALUES
 ('botella','package',true),('botella','box',false),
 ('lata','package',true),('lata','box',false),
 ('und','package',false),('und','box',false),('und','bag',false),
 ('kg','bag',false),('kg','sack',true),('g','bag',false),
 ('l','drum',true),('ml','drum',false),
 ('bolsa','package',true),('bolsa','box',false),('paquete','box',true);
-- Package and box were valid for all units in earlier clients. Keep those
-- combinations available; preferred pairings above remain unchanged.
INSERT INTO inventory_unit_combination_templates(unit,presentation_type)
 SELECT u.code,p.code FROM inventory_unit_templates u
 CROSS JOIN inventory_presentation_type_templates p WHERE p.code IN ('package','box')
 ON CONFLICT(unit,presentation_type) DO NOTHING;
CREATE TABLE inventory_unit_combinations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  unit text NOT NULL,
  presentation_type text NOT NULL,
  is_default boolean NOT NULL DEFAULT false,
  UNIQUE(organization_id,unit,presentation_type),
  FOREIGN KEY(organization_id,unit) REFERENCES inventory_units(organization_id,code),
  FOREIGN KEY(organization_id,presentation_type) REFERENCES inventory_presentation_types(organization_id,code)
);
CREATE UNIQUE INDEX inventory_unit_combination_default_uq ON inventory_unit_combinations(organization_id,unit) WHERE is_default;
INSERT INTO inventory_presentation_types(organization_id,code,name,sort_order)
 SELECT o.id,t.code,t.name,t.sort_order FROM organizations o CROSS JOIN inventory_presentation_type_templates t;
INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type,is_default)
 SELECT u.organization_id,u.code,t.presentation_type,t.is_default FROM inventory_units u JOIN inventory_unit_combination_templates t ON t.unit=u.code;
INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type)
 SELECT u.organization_id,u.code,p.code FROM inventory_units u
 JOIN inventory_presentation_types p ON p.organization_id=u.organization_id AND p.code IN ('package','box')
 ON CONFLICT(organization_id,unit,presentation_type) DO NOTHING;
-- Preserve every previously registered combination, including custom base units.
INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type)
 SELECT DISTINCT i.organization_id,i.unit,p.presentation_type FROM inventory_items i
 JOIN inventory_presentations p ON p.organization_id=i.organization_id AND p.inventory_item_id=i.id
 ON CONFLICT(organization_id,unit,presentation_type) DO NOTHING;
CREATE FUNCTION seed_inventory_combinations() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO inventory_presentation_types(organization_id,code,name,sort_order)
  SELECT NEW.id,code,name,sort_order FROM inventory_presentation_type_templates;
 INSERT INTO inventory_unit_combinations(organization_id,unit,presentation_type,is_default)
  SELECT u.organization_id,u.code,t.presentation_type,t.is_default FROM inventory_units u
  JOIN inventory_unit_combination_templates t ON t.unit=u.code WHERE u.organization_id=NEW.id;
 RETURN NEW;
END;
$$;
-- The unit seed trigger runs first (alphabetical PostgreSQL trigger ordering).
CREATE TRIGGER organization_inventory_zcombinations AFTER INSERT ON organizations
 FOR EACH ROW EXECUTE FUNCTION seed_inventory_combinations();
ALTER TABLE inventory_presentations DROP CONSTRAINT inventory_presentations_presentation_type_check,
 DROP CONSTRAINT inventory_presentations_factor_check,
 ADD CONSTRAINT inventory_presentations_type_fk FOREIGN KEY(organization_id,presentation_type) REFERENCES inventory_presentation_types(organization_id,code),
 ADD CONSTRAINT inventory_presentations_factor_check CHECK((presentation_type='unit' AND units_per_presentation=1) OR (presentation_type<>'unit' AND units_per_presentation>0)),
 ADD COLUMN is_default boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX inventory_presentation_default_uq ON inventory_presentations(organization_id,inventory_item_id) WHERE is_default;
UPDATE inventory_presentations p SET is_default=true WHERE p.id IN (
 SELECT DISTINCT ON(organization_id,inventory_item_id) id FROM inventory_presentations WHERE active
 ORDER BY organization_id,inventory_item_id,(presentation_type<>'unit') DESC,created_at DESC,id
);
ALTER TABLE inventory_entries DROP CONSTRAINT inventory_entries_presentation_type_check,
 DROP CONSTRAINT inventory_entries_presentation_factor_check,
 ADD CONSTRAINT inventory_entries_type_fk FOREIGN KEY(organization_id,presentation_type) REFERENCES inventory_presentation_types(organization_id,code),
 ADD CONSTRAINT inventory_entries_presentation_factor_check CHECK((presentation_type='unit' AND units_per_presentation=1) OR (presentation_type<>'unit' AND units_per_presentation>0));
ALTER TABLE purchase_order_items DROP CONSTRAINT purchase_order_items_presentation_type_check,
 ADD CONSTRAINT purchase_order_items_type_fk FOREIGN KEY(organization_id,presentation_type) REFERENCES inventory_presentation_types(organization_id,code);
ALTER TABLE purchase_receipt_items DROP CONSTRAINT purchase_receipt_items_presentation_type_check,
 ADD CONSTRAINT purchase_receipt_items_type_fk FOREIGN KEY(organization_id,presentation_type) REFERENCES inventory_presentation_types(organization_id,code);
ALTER TABLE purchase_return_items DROP CONSTRAINT purchase_return_items_presentation_type_check,
 ADD CONSTRAINT purchase_return_items_type_fk FOREIGN KEY(organization_id,presentation_type) REFERENCES inventory_presentation_types(organization_id,code);
