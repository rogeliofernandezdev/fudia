-- Rol › Menú refleja el menú lateral: una clave por pantalla. Antes una clave abría
-- varias pantallas (dashboard → Dashboard y Ventas, facturacion → Comprobantes y
-- Facturación, pedidos → Pedidos y Salón, locales → Locales y Mi restaurante,
-- fiscal → Fiscal y moneda y Medios de pago). Cada rol conserva lo que ya veía:
-- recibe la clave nueva solo si tenía la anterior y la acción que esa pantalla exige.
UPDATE roles SET menu_access = menu_access || ARRAY['ventas']
WHERE 'dashboard' = ANY(menu_access) AND NOT 'ventas' = ANY(menu_access)
  AND permissions && ARRAY['orders.read','*'];

UPDATE roles SET menu_access = menu_access || ARRAY['comprobantes']
WHERE 'facturacion' = ANY(menu_access) AND NOT 'comprobantes' = ANY(menu_access)
  AND permissions && ARRAY['orders.read','*'];

UPDATE roles SET menu_access = menu_access || ARRAY['salon']
WHERE 'pedidos' = ANY(menu_access) AND NOT 'salon' = ANY(menu_access)
  AND permissions && ARRAY['orders.read','*'];

UPDATE roles SET menu_access = menu_access || ARRAY['mi_restaurante']
WHERE 'locales' = ANY(menu_access) AND NOT 'mi_restaurante' = ANY(menu_access)
  AND permissions && ARRAY['organizations.read','*'];

UPDATE roles SET menu_access = menu_access || ARRAY['medios_pago']
WHERE 'fiscal' = ANY(menu_access) AND NOT 'medios_pago' = ANY(menu_access)
  AND permissions && ARRAY['organizations.read','*'];
