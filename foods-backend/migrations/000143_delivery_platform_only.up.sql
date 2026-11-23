-- Delivery lo administra Fudia (2026-11-23): la opción «Delivery» del menú es
-- solo de Plataforma (centrales, tarifas de envío por local y entregas
-- fallidas). Se retira la casilla de los roles de empresa; no cambia ningún
-- permiso ni plan. Los módulos delivery, delivery_apps y repartidores salen del
-- catálogo de Módulos: Delivery Fudia se activa por empresa en Plataforma.
UPDATE roles
SET menu_access=array_remove(menu_access,'delivery')
WHERE 'delivery'=ANY(menu_access);
