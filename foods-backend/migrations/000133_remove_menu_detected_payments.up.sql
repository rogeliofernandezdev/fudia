-- «Pagos detectados» volvió a ser una pestaña de Caja: su casilla de menú deja
-- de existir. Limpia la clave en bases donde se llegó a otorgar.
UPDATE roles
SET menu_access=array_remove(menu_access,'pagos_detectados')
WHERE 'pagos_detectados'=ANY(menu_access);
