-- Never convert or discard custom packaging/history to force a downgrade.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM inventory_presentations WHERE presentation_type NOT IN ('unit','package','box') OR (presentation_type<>'unit' AND units_per_presentation<=1)) THEN
  RAISE EXCEPTION 'Cannot downgrade: custom presentations or fractional conversions are in use. Preserve migration 77.';
 END IF;
END $$;
ALTER TABLE purchase_return_items DROP CONSTRAINT purchase_return_items_type_fk,
 ADD CONSTRAINT purchase_return_items_presentation_type_check CHECK(presentation_type IN ('unit','package','box'));
ALTER TABLE purchase_receipt_items DROP CONSTRAINT purchase_receipt_items_type_fk,
 ADD CONSTRAINT purchase_receipt_items_presentation_type_check CHECK(presentation_type IN ('unit','package','box'));
ALTER TABLE purchase_order_items DROP CONSTRAINT purchase_order_items_type_fk,
 ADD CONSTRAINT purchase_order_items_presentation_type_check CHECK(presentation_type IN ('unit','package','box'));
ALTER TABLE inventory_entries DROP CONSTRAINT inventory_entries_type_fk,DROP CONSTRAINT inventory_entries_presentation_factor_check,
 ADD CONSTRAINT inventory_entries_presentation_type_check CHECK(presentation_type IN ('unit','package','box')),
 ADD CONSTRAINT inventory_entries_presentation_factor_check CHECK((presentation_type='unit' AND units_per_presentation=1) OR (presentation_type IN ('package','box') AND units_per_presentation>1));
ALTER TABLE inventory_presentations DROP CONSTRAINT inventory_presentations_type_fk,DROP CONSTRAINT inventory_presentations_factor_check,
 DROP COLUMN is_default,
 ADD CONSTRAINT inventory_presentations_presentation_type_check CHECK(presentation_type IN ('unit','package','box')),
 ADD CONSTRAINT inventory_presentations_factor_check CHECK((presentation_type='unit' AND units_per_presentation=1) OR (presentation_type IN ('package','box') AND units_per_presentation>1));
DROP TRIGGER organization_inventory_zcombinations ON organizations;
DROP FUNCTION seed_inventory_combinations();
DROP TABLE inventory_unit_combinations;
DROP TABLE inventory_unit_combination_templates;
DROP TABLE inventory_presentation_types;
DROP TABLE inventory_presentation_type_templates;
