-- Conversion metadata is a database catalog. Organization units retain their identity.
CREATE TABLE inventory_measurement_units (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  family text NOT NULL CHECK(family IN ('mass','volume','count')),
  family_label text NOT NULL,
  reference_factor numeric(20,9) NOT NULL CHECK(reference_factor>0),
  inventory_default boolean NOT NULL DEFAULT false,
  recipe_default boolean NOT NULL DEFAULT false
);
INSERT INTO inventory_measurement_units(code,family,family_label,reference_factor,inventory_default,recipe_default) VALUES
 ('kg','mass','Peso',1000,true,false),('g','mass','Peso',1,false,true),
 ('l','volume','Volumen',1000,true,false),('ml','volume','Volumen',1,false,true),
 ('und','count','Unidades',1,true,true),
 ('botella','count','Unidades',1,false,false),('lata','count','Unidades',1,false,false),
 ('paquete','count','Unidades',1,false,false),('bolsa','count','Unidades',1,false,false),('caja','count','Unidades',1,false,false);

ALTER TABLE product_recipe_items
 ALTER COLUMN quantity TYPE numeric(17,6),
 ADD COLUMN consumption_unit text,
 ADD COLUMN consumption_quantity numeric(20,6),
 ADD COLUMN consumption_factor numeric(20,9),
 ADD CONSTRAINT recipe_consumption_unit_fk FOREIGN KEY(organization_id,consumption_unit) REFERENCES inventory_units(organization_id,code),
 ADD CONSTRAINT recipe_consumption_equivalence CHECK (
   (consumption_unit IS NULL AND consumption_quantity IS NULL AND consumption_factor IS NULL) OR
   (consumption_unit IS NOT NULL AND consumption_quantity IS NOT NULL AND consumption_factor IS NOT NULL
    AND consumption_quantity>0 AND consumption_factor>0 AND quantity=round(consumption_quantity*consumption_factor,6))
 );
UPDATE product_recipe_items r SET consumption_unit=i.unit,consumption_quantity=r.quantity,consumption_factor=1
 FROM inventory_items i WHERE i.id=r.inventory_item_id AND i.organization_id=r.organization_id;

-- Preserve sub-gram/litre quantities through consumption, reversal and later stock documents.
ALTER TABLE stock_balances ALTER COLUMN quantity TYPE numeric(17,6);
ALTER TABLE stock_movements ALTER COLUMN quantity_delta TYPE numeric(17,6), ALTER COLUMN balance_after TYPE numeric(17,6);
ALTER TABLE inventory_adjustments ALTER COLUMN stock_before TYPE numeric(17,6), ALTER COLUMN stock_after TYPE numeric(17,6);
