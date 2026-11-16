UPDATE roles
SET menu_access=array_remove(menu_access,'pagos_detectados')
WHERE 'pagos_detectados'=ANY(menu_access);
