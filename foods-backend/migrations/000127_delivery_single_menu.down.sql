-- Los roles que tenían Delivery recuperan las dos opciones que existían antes.
UPDATE roles
SET menu_access=menu_access||ARRAY['delivery_apps','repartidores']::text[]
WHERE 'delivery'=ANY(menu_access) AND NOT menu_access && ARRAY['delivery_apps','repartidores']::text[];
