DO $$
DECLARE
  platform_count integer;
BEGIN
  SELECT count(*) INTO platform_count FROM users WHERE platform_admin;
  IF platform_count > 1 THEN
    RAISE EXCEPTION 'FUDIA requires a single platform_admin account; found %', platform_count;
  END IF;
  IF platform_count = 0 THEN
    UPDATE users
    SET platform_admin=true,updated_at=now()
    WHERE id=(
      SELECT id FROM users
      WHERE lower(email)='admin@foods.local' AND active
      ORDER BY created_at,id
      LIMIT 1
    );
  END IF;
END
$$;

CREATE UNIQUE INDEX users_single_platform_admin_uq
  ON users ((platform_admin))
  WHERE platform_admin;

UPDATE roles
SET name='Administrador de empresa',
    description='Administra la empresa, sus locales, equipo y operación sin privilegios de plataforma',
    menu_access=ARRAY[
      'dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','recetas',
      'inventario','kardex','compras','logistica','clientes','locales','fiscal','usuarios','call_center','carta_qr','kiosco',
      'delivery','delivery_apps','repartidores','crm','puntos','ofertas','personal','costos','bi','app_manager','facturacion',
      'integraciones','whatsapp_bot'
    ]::text[],
    permissions=ARRAY[
      'dashboard.read','users.read','users.manage','organizations.read','organizations.manage','fiscal.read',
      'menu.read','menu.manage','recipes.manage','customers.read','customers.manage',
      'orders.read','orders.manage','kitchen.manage','tables.read','tables.manage','reservations.read','reservations.manage',
      'cash.read','cash.manage','cash.expected.read','receipts.read','receipts.manage',
      'inventory.read','inventory.manage','inventory.transfer',
      'purchases.read','purchases.manage','purchases.approve','purchases.receive',
      'reports.read','audit.read','delivery.read','delivery.manage'
    ]::text[],
    active=true,
    updated_at=now()
WHERE system_key='administrator';

UPDATE roles
SET permissions=ARRAY[
      'dashboard.read','users.read','users.manage','organizations.read','organizations.manage','fiscal.read',
      'menu.read','menu.manage','recipes.manage','customers.read','customers.manage',
      'orders.read','orders.manage','kitchen.manage','tables.read','tables.manage','reservations.read','reservations.manage',
      'cash.read','cash.manage','cash.expected.read','receipts.read','receipts.manage',
      'inventory.read','inventory.manage','inventory.transfer',
      'purchases.read','purchases.manage','purchases.approve','purchases.receive',
      'reports.read','audit.read','delivery.read','delivery.manage'
    ]::text[],
    updated_at=now()
WHERE system_key IS DISTINCT FROM 'administrator'
  AND '*'=ANY(permissions);

UPDATE roles
SET menu_access=ARRAY[
      'dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','recetas',
      'inventario','kardex','compras','logistica','clientes','locales','fiscal','usuarios','call_center','carta_qr','kiosco',
      'delivery','delivery_apps','repartidores','crm','puntos','ofertas','personal','costos','bi','app_manager','facturacion',
      'integraciones','whatsapp_bot'
    ]::text[],
    updated_at=now()
WHERE '*'=ANY(menu_access);

ALTER TABLE roles
  ADD CONSTRAINT roles_permissions_no_wildcard CHECK (NOT ('*'=ANY(permissions))),
  ADD CONSTRAINT roles_menu_access_no_wildcard CHECK (NOT ('*'=ANY(menu_access)));
