-- Las unidades contables (se cuentan por piezas) no admiten presentaciones con
-- menos de una pieza: un paquete de 0.001 botellas no existe. Las unidades de
-- medida (kg, g, l, ml) y las creadas por cada empresa siguen admitiendo fracciones.
ALTER TABLE inventory_unit_templates ADD COLUMN countable boolean NOT NULL DEFAULT false;
ALTER TABLE inventory_units ADD COLUMN countable boolean NOT NULL DEFAULT false;

UPDATE inventory_unit_templates SET countable=true
WHERE code IN ('und','botella','lata','paquete','bolsa','caja');

-- Solo clasifica el catálogo; no modifica stock, presentaciones ni historial.
UPDATE inventory_units SET countable=true
WHERE code IN ('und','botella','lata','paquete','bolsa','caja');

CREATE OR REPLACE FUNCTION seed_inventory_units() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO inventory_units(organization_id,code,name,sort_order,countable)
  SELECT NEW.id,code,name,sort_order,countable FROM inventory_unit_templates;
  RETURN NEW;
END;
$$;
