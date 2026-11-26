-- Devuelve Reservas al Mesero y se la retira al Cajero. tables.read se conserva
-- en el Cajero: ya lo tenía por Mesas y zonas.
UPDATE roles
SET description='Gestiona mesas, clientes, pedidos y reservas',
    menu_access=CASE WHEN 'reservas'=ANY(menu_access) THEN menu_access ELSE array_append(menu_access,'reservas') END,
    permissions=permissions||ARRAY(
      SELECT value
      FROM unnest(ARRAY['reservations.read','reservations.manage']::text[]) value
      WHERE NOT (value=ANY(permissions))
    ),
    updated_at=now()
WHERE system_key='waiter' AND NOT ('*'=ANY(permissions));

UPDATE roles
SET description='Opera caja, cobros y arqueos',
    menu_access=array_remove(menu_access,'reservas'),
    permissions=array_remove(array_remove(permissions,'reservations.read'),'reservations.manage'),
    updated_at=now()
WHERE system_key='cashier' AND NOT ('*'=ANY(permissions));
