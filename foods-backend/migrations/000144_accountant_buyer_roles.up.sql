-- Roles predeterminados con nombre de persona (2026-11-23): «Contabilidad» pasa a
-- «Contador» y «Compras» a «Comprador», como Cajero, Mesero o Almacenero.
-- Solo se renombra si conserva el nombre original y la empresa no tiene ya otro
-- rol con el nombre nuevo (roles es UNIQUE(organization_id,name)).
UPDATE roles r
SET name='Contador',updated_at=now()
WHERE r.system_key='accounting' AND r.name='Contabilidad'
  AND NOT EXISTS (SELECT 1 FROM roles o WHERE o.organization_id=r.organization_id AND o.name='Contador');

UPDATE roles r
SET name='Comprador',updated_at=now()
WHERE r.system_key='buyer' AND r.name='Compras'
  AND NOT EXISTS (SELECT 1 FROM roles o WHERE o.organization_id=r.organization_id AND o.name='Comprador');

-- El contador consulta Ventas y Comprobantes en solo lectura: orders.read sin
-- orders.manage. El catálogo efectivo los oculta si el plan no los incluye.
UPDATE roles
SET description='Consulta ventas, comprobantes, cierres de caja, gastos e información fiscal',
    menu_access=array_cat(
      menu_access,
      ARRAY(
        SELECT value
        FROM unnest(ARRAY['ventas','comprobantes']::text[]) value
        WHERE NOT (value=ANY(menu_access))
      )
    ),
    permissions=CASE WHEN 'orders.read'=ANY(permissions) THEN permissions ELSE array_append(permissions,'orders.read') END,
    updated_at=now()
WHERE system_key='accounting';
