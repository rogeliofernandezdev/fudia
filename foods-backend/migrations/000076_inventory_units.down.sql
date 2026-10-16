ALTER TABLE inventory_items DROP CONSTRAINT inventory_items_unit_fk;
DROP TRIGGER organization_inventory_units ON organizations;
DROP FUNCTION seed_inventory_units();
DROP TABLE inventory_units;
DROP TABLE inventory_unit_templates;
