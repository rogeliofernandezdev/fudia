CREATE TABLE expense_category_templates (
  name text PRIMARY KEY,
  active boolean NOT NULL DEFAULT true
);

INSERT INTO expense_category_templates(name,active) VALUES
  ('Alquiler',true),
  ('Servicios',true),
  ('Mantenimiento',true),
  ('Limpieza',true),
  ('Transporte',true),
  ('Otros',true);

CREATE TABLE location_zone_templates (
  name text PRIMARY KEY,
  sort_order integer NOT NULL DEFAULT 0,
  active boolean NOT NULL DEFAULT true
);

INSERT INTO location_zone_templates(name,sort_order,active) VALUES
  ('Principal',10,true);

CREATE TABLE cash_register_templates (
  name text PRIMARY KEY,
  blind_close boolean NOT NULL DEFAULT false,
  active boolean NOT NULL DEFAULT true
);

INSERT INTO cash_register_templates(name,blind_close,active) VALUES
  ('Caja principal',false,true);
