-- Reservas (2026-11-26): las atiende el Cajero, no el Mesero. Se aplica a los
-- roles predeterminados de cada empresa (system_key). El catálogo efectivo
-- sigue ocultando Reservas si el plan de la empresa no la incluye.
UPDATE roles
SET description='Opera caja, cobros, arqueos y reservas',
    menu_access=CASE WHEN 'reservas'=ANY(menu_access) THEN menu_access ELSE array_append(menu_access,'reservas') END,
    permissions=permissions||ARRAY(
      SELECT value
      FROM unnest(ARRAY['reservations.read','reservations.manage','tables.read']::text[]) value
      WHERE NOT (value=ANY(permissions))
    ),
    updated_at=now()
WHERE system_key='cashier' AND NOT ('*'=ANY(permissions));

UPDATE roles
SET description='Gestiona mesas, clientes y pedidos',
    menu_access=array_remove(menu_access,'reservas'),
    permissions=array_remove(array_remove(permissions,'reservations.read'),'reservations.manage'),
    updated_at=now()
WHERE system_key='waiter' AND NOT ('*'=ANY(permissions));
