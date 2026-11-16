-- «Pagos detectados» deja de ser una pestaña de Caja y pasa a ser su propia
-- opción del menú (Operación), con casilla propia en Usuarios y roles › Menú.
-- Todo rol que ya veía Caja conserva el acceso a los pagos detectados.
UPDATE roles
SET menu_access=array_append(menu_access,'pagos_detectados')
WHERE 'caja'=ANY(menu_access) AND NOT 'pagos_detectados'=ANY(menu_access);
