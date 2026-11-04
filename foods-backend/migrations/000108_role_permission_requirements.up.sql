-- Roles personalizados existentes: completa las acciones que exigen sus pantallas
-- (Menú) y acciones marcadas, igual que SaveRole al guardarlos (role_requirements.go).
-- Una acción fuera del plan no tiene efecto: el catálogo efectivo la filtra.
CREATE TEMP TABLE role_menu_requirements(menu text, permission text);
INSERT INTO role_menu_requirements VALUES
  ('dashboard','dashboard.read'),
  ('pos','orders.read'),
  ('pedidos','orders.read'),
  ('pedidos','menu.read'),
  ('cocina','orders.read'),
  ('mesas','tables.read'),
  ('caja','cash.read'),
  ('reservas','reservations.read'),
  ('productos','menu.read'),
  ('disponibilidad','menu.read'),
  ('combos','menu.read'),
  ('recetas','menu.read'),
  ('recetas','inventory.read'),
  ('inventario','inventory.read'),
  ('kardex','inventory.read'),
  ('compras','purchases.read'),
  ('clientes','customers.read'),
  ('locales','organizations.read'),
  ('costos','expenses.read'),
  ('fiscal','organizations.read'),
  ('usuarios','users.read'),
  ('usuarios','organizations.read'),
  ('facturacion','organizations.read'),
  ('whatsapp_bot','organizations.read');
CREATE TEMP TABLE role_permission_requirements(permission text, required text);
INSERT INTO role_permission_requirements VALUES
  ('users.read','organizations.read'),
  ('users.manage','users.read'),
  ('users.manage','organizations.read'),
  ('organizations.manage','organizations.read'),
  ('menu.manage','menu.read'),
  ('recipes.manage','menu.read'),
  ('recipes.manage','inventory.read'),
  ('customers.manage','customers.read'),
  ('orders.manage','orders.read'),
  ('orders.manage','menu.read'),
  ('kitchen.manage','orders.read'),
  ('bar.manage','orders.read'),
  ('tables.manage','tables.read'),
  ('reservations.manage','reservations.read'),
  ('cash.manage','cash.read'),
  ('cash.manage','orders.read'),
  ('cash.registers.manage','cash.read'),
  ('cash.expected.read','cash.read'),
  ('receipts.manage','receipts.read'),
  ('inventory.manage','inventory.read'),
  ('inventory.transfer','inventory.read'),
  ('purchases.manage','purchases.read'),
  ('purchases.approve','purchases.read'),
  ('purchases.receive','purchases.read'),
  ('expenses.manage','expenses.read'),
  ('expenses.void','expenses.read'),
  ('delivery.manage','delivery.read');

UPDATE roles r SET permissions=r.permissions||missing.items
FROM (
  SELECT r2.id,array_agg(DISTINCT q.permission) AS items
  FROM roles r2 JOIN role_menu_requirements q ON q.menu=ANY(r2.menu_access)
  WHERE NOT ('*'=ANY(r2.permissions)) AND NOT (q.permission=ANY(r2.permissions))
  GROUP BY r2.id
) missing
WHERE r.id=missing.id;

-- Requisitos entre acciones (transitivos: a lo sumo dos niveles).
DO $$
BEGIN
  FOR i IN 1..3 LOOP
    UPDATE roles r SET permissions=r.permissions||missing.items
    FROM (
      SELECT r2.id,array_agg(DISTINCT q.required) AS items
      FROM roles r2 JOIN role_permission_requirements q ON q.permission=ANY(r2.permissions)
      WHERE NOT ('*'=ANY(r2.permissions)) AND NOT (q.required=ANY(r2.permissions))
      GROUP BY r2.id
    ) missing
    WHERE r.id=missing.id;
  END LOOP;
END $$;

DROP TABLE role_menu_requirements;
DROP TABLE role_permission_requirements;
