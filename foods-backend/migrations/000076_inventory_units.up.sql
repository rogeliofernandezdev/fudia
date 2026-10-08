CREATE TABLE inventory_unit_templates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  name text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0
);

INSERT INTO inventory_unit_templates(code,name,sort_order) VALUES
 ('und','Unidad',10),('botella','Botella',20),('lata','Lata',30),
 ('paquete','Paquete',40),('bolsa','Bolsa',50),('caja','Caja',60),
 ('kg','Kilogramo',70),('g','Gramo',80),('l','Litro',90),('ml','Mililitro',100);

CREATE TABLE inventory_units (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  code text NOT NULL,
  name text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(organization_id,code)
);

INSERT INTO inventory_units(organization_id,code,name,sort_order)
SELECT o.id,t.code,t.name,t.sort_order FROM organizations o CROSS JOIN inventory_unit_templates t;

-- Preserve every previously used code without converting stock or history.
INSERT INTO inventory_units(organization_id,code,name,sort_order)
SELECT DISTINCT organization_id,unit,unit,1000 FROM inventory_items
ON CONFLICT(organization_id,code) DO NOTHING;

CREATE FUNCTION seed_inventory_units() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO inventory_units(organization_id,code,name,sort_order)
  SELECT NEW.id,code,name,sort_order FROM inventory_unit_templates;
  RETURN NEW;
END;
$$;
CREATE TRIGGER organization_inventory_units AFTER INSERT ON organizations
FOR EACH ROW EXECUTE FUNCTION seed_inventory_units();

ALTER TABLE inventory_items ADD CONSTRAINT inventory_items_unit_fk
FOREIGN KEY(organization_id,unit) REFERENCES inventory_units(organization_id,code);
