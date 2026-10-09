-- Endurece identidad para tenants creados después de las migraciones de roles.
-- Repara el Administrador y completa roles predeterminados faltantes sin alterar roles personalizados existentes.

UPDATE roles r
SET system_key='administrator',
    description=CASE WHEN btrim(description)='' THEN 'Acceso total a la empresa y sus locales' ELSE description END,
    menu_access=ARRAY['*']::text[],
    permissions=ARRAY['*']::text[],
    active=true,
    updated_at=now()
WHERE r.name='Administrador'
  AND r.system_key IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM roles existing
    WHERE existing.organization_id=r.organization_id
      AND existing.system_key='administrator'
  );

UPDATE roles
SET menu_access=ARRAY['*']::text[],
    permissions=ARRAY['*']::text[],
    active=true,
    updated_at=now()
WHERE system_key='administrator';

INSERT INTO roles(organization_id,name,system_key,description,menu_access,permissions)
SELECT o.id,v.name,v.key,v.description,v.menu_access,v.permissions
FROM organizations o
CROSS JOIN (VALUES
 ('Gerente de local','location_manager','Gestiona la operación completa de los locales asignados',
  ARRAY['dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','recetas','inventario','kardex','compras','clientes','locales','costos','bi','fiscal','usuarios','facturacion','integraciones']::text[],
  ARRAY['dashboard.read','menu.read','menu.manage','customers.read','customers.manage','orders.read','orders.manage','cash.read','cash.manage','inventory.read','purchases.read','reports.read']::text[]),
 ('Supervisor de turno','shift_supervisor','Supervisa atención, comandas, mesas, caja y turnos',
  ARRAY['dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','clientes']::text[],
  ARRAY['dashboard.read','menu.read','customers.read','customers.manage','orders.read','orders.manage','cash.read','cash.manage','tables.read','tables.manage']::text[]),
 ('Cajero','cashier','Opera caja, cobros, comprobantes y arqueos',
  ARRAY['pos','pedidos','mesas','caja','clientes','facturacion']::text[],
  ARRAY['menu.read','customers.read','customers.manage','orders.read','orders.manage','cash.read','cash.manage','receipts.manage']::text[]),
 ('Mesero','waiter','Gestiona mesas, clientes, pedidos y comandas',
  ARRAY['pos','pedidos','mesas']::text[],
  ARRAY['menu.read','customers.read','customers.manage','orders.read','orders.manage','tables.read']::text[]),
 ('Cocinero','cook','Consulta y actualiza comandas de cocina',
  ARRAY['cocina','disponibilidad']::text[],
  ARRAY['menu.read','orders.read','kitchen.manage']::text[]),
 ('Jefe de cocina','kitchen_manager','Gestiona comandas, disponibilidad, recetas y producción',
  ARRAY['cocina','productos','disponibilidad','recetas','inventario']::text[],
  ARRAY['menu.read','menu.manage','orders.read','kitchen.manage','recipes.manage','inventory.read']::text[]),
 ('Almacenero','warehouse','Gestiona inventario, kardex y recepción',
  ARRAY['inventario','kardex','compras']::text[],
  ARRAY['inventory.read','inventory.manage','inventory.transfer','purchases.read','purchases.receive']::text[]),
 ('Compras','buyer','Gestiona proveedores y órdenes de compra',
  ARRAY['inventario','kardex','compras','logistica']::text[],
  ARRAY['inventory.read','purchases.read','purchases.manage','purchases.approve']::text[]),
 ('Repartidor','courier','Consulta pedidos asignados y actualiza entregas',
  ARRAY['delivery']::text[],
  ARRAY['delivery.read','delivery.manage']::text[]),
 ('Contabilidad','accounting','Consulta cierres, comprobantes, impuestos y reportes',
  ARRAY['dashboard','caja','costos','bi','fiscal','facturacion']::text[],
  ARRAY['cash.read','receipts.read','reports.read','fiscal.read']::text[]),
 ('Auditor','auditor','Acceso de solo lectura a operación e historial',
  ARRAY['dashboard','productos','clientes','inventario','kardex','compras','costos','bi']::text[],
  ARRAY['dashboard.read','menu.read','customers.read','orders.read','cash.read','inventory.read','purchases.read','reports.read','audit.read']::text[])
) AS v(name,key,description,menu_access,permissions)
WHERE NOT EXISTS (
  SELECT 1 FROM roles r
  WHERE r.organization_id=o.id AND (r.system_key=v.key OR r.name=v.name)
);
