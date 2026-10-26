CREATE OR REPLACE FUNCTION seed_inventory_units() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO inventory_units(organization_id,code,name,sort_order)
  SELECT NEW.id,code,name,sort_order FROM inventory_unit_templates;
  RETURN NEW;
END;
$$;

ALTER TABLE inventory_units DROP COLUMN countable;
ALTER TABLE inventory_unit_templates DROP COLUMN countable;
