-- «Costos y gastos» pasa a ser la pestaña «Gastos» de Caja (2026-11-22): su
-- casilla de menú deja de existir. Los roles que la tenían conservan el acceso
-- a los gastos por Caja cuando ya podían ver Caja (cash.read); nunca se
-- otorga un permiso nuevo. Los permisos expenses.* no cambian.
UPDATE roles
SET menu_access=CASE
      WHEN 'caja'=ANY(menu_access) OR NOT ('cash.read'=ANY(permissions) OR '*'=ANY(permissions))
        THEN array_remove(menu_access,'costos')
      ELSE array_append(array_remove(menu_access,'costos'),'caja')
    END
WHERE 'costos'=ANY(menu_access);
