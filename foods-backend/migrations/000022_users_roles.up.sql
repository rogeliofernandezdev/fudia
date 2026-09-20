ALTER TABLE roles ADD COLUMN system_key text;
ALTER TABLE roles ADD COLUMN description text NOT NULL DEFAULT '';
ALTER TABLE roles ADD COLUMN active boolean NOT NULL DEFAULT true;
ALTER TABLE roles ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE roles ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();
CREATE UNIQUE INDEX roles_system_key_uq ON roles(organization_id,system_key) WHERE system_key IS NOT NULL;

UPDATE roles SET system_key='administrator',description='Acceso total a la empresa y sus locales' WHERE name='Administrador' AND system_key IS NULL;

INSERT INTO roles(organization_id,name,system_key,description,permissions)
SELECT o.id,v.name,v.key,v.description,v.permissions
FROM organizations o CROSS JOIN (VALUES
 ('Gerente de local','location_manager','Gestiona la operación completa de los locales asignados',ARRAY['dashboard.read','menu.read','menu.manage','customers.read','customers.manage','orders.read','orders.manage','cash.read','cash.manage','inventory.read','purchases.read','reports.read']::text[]),
 ('Supervisor de turno','shift_supervisor','Supervisa atención, comandas, mesas, caja y turnos',ARRAY['dashboard.read','menu.read','customers.read','customers.manage','orders.read','orders.manage','cash.read','cash.manage','tables.read','tables.manage']::text[]),
 ('Cajero','cashier','Opera caja, cobros, comprobantes y arqueos',ARRAY['menu.read','customers.read','customers.manage','orders.read','orders.manage','cash.read','cash.manage','receipts.manage']::text[]),
 ('Mesero','waiter','Gestiona mesas, clientes, pedidos y comandas',ARRAY['menu.read','customers.read','customers.manage','orders.read','orders.manage','tables.read']::text[]),
 ('Cocinero','cook','Consulta y actualiza comandas de cocina',ARRAY['menu.read','orders.read','kitchen.manage']::text[]),
 ('Jefe de cocina','kitchen_manager','Gestiona comandas, disponibilidad, recetas y producción',ARRAY['menu.read','menu.manage','orders.read','kitchen.manage','recipes.manage','inventory.read']::text[]),
 ('Almacenero','warehouse','Gestiona inventario, kardex y recepción',ARRAY['inventory.read','inventory.manage','purchases.read','purchases.receive']::text[]),
 ('Compras','buyer','Gestiona proveedores y órdenes de compra',ARRAY['inventory.read','purchases.read','purchases.manage']::text[]),
 ('Repartidor','courier','Consulta pedidos asignados y actualiza entregas',ARRAY['delivery.read','delivery.manage']::text[]),
 ('Contabilidad','accounting','Consulta cierres, comprobantes, impuestos y reportes',ARRAY['cash.read','receipts.read','reports.read','fiscal.read']::text[]),
 ('Auditor','auditor','Acceso de solo lectura a operación e historial',ARRAY['dashboard.read','menu.read','customers.read','orders.read','cash.read','inventory.read','purchases.read','reports.read','audit.read']::text[])
) AS v(name,key,description,permissions)
ON CONFLICT (organization_id,name) DO NOTHING;

CREATE INDEX user_roles_role_idx ON user_roles(role_id,location_id);
