-- El menú del restaurante tiene una sola opción «Delivery» (clave delivery) con
-- pestañas. «Apps de delivery» y «App repartidores» dejan de ser opciones del
-- menú: sus pestañas dependen solo del módulo activo. Se quitan sus claves de los
-- roles para que el catálogo de menús y los roles guardados coincidan.
UPDATE roles
SET menu_access=array_remove(array_remove(menu_access,'delivery_apps'),'repartidores')
WHERE menu_access && ARRAY['delivery_apps','repartidores']::text[];
