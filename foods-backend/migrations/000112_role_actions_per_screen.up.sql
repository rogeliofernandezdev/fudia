-- Rol › Acciones muestra solo lo que cada pantalla realmente habilita.
-- 1. Estas acciones no se validan en ninguna ruta ni pantalla: marcarlas no
--    concedía nada. Comprobantes usa orders.*, Fiscal y moneda organizations.* y
--    Delivery aún no tiene pantallas. Se retiran de los roles.
UPDATE roles SET permissions = ARRAY(
  SELECT p FROM unnest(permissions) AS p
  WHERE p NOT IN ('reports.read','audit.read','receipts.read','receipts.manage','fiscal.read','delivery.read','delivery.manage'))
WHERE permissions && ARRAY['reports.read','audit.read','receipts.read','receipts.manage','fiscal.read','delivery.read','delivery.manage'];

-- 2. Requisitos que faltaban (role_requirements.go): Mesas y zonas lista las
--    zonas con menu.read y Reservas asigna mesas con tables.read. Igual que al
--    guardar el rol, se completan en los roles que tienen la pantalla.
UPDATE roles SET permissions = permissions || ARRAY['menu.read']
WHERE 'mesas' = ANY(menu_access) AND NOT permissions && ARRAY['menu.read','*'];

UPDATE roles SET permissions = permissions || ARRAY['tables.read']
WHERE 'reservas' = ANY(menu_access) AND NOT permissions && ARRAY['tables.read','*'];
