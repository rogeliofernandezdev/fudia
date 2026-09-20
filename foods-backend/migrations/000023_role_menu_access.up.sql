ALTER TABLE roles ADD COLUMN menu_access text[] NOT NULL DEFAULT '{}';

UPDATE roles SET menu_access = CASE system_key
  WHEN 'administrator' THEN ARRAY['*']::text[]
  WHEN 'location_manager' THEN ARRAY['dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','recetas','inventario','kardex','compras','clientes','locales','costos','bi','fiscal','usuarios','facturacion','integraciones']::text[]
  WHEN 'shift_supervisor' THEN ARRAY['dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','clientes']::text[]
  WHEN 'cashier' THEN ARRAY['pos','pedidos','mesas','caja','clientes','facturacion']::text[]
  WHEN 'waiter' THEN ARRAY['pos','pedidos','mesas']::text[]
  WHEN 'cook' THEN ARRAY['cocina','disponibilidad']::text[]
  WHEN 'kitchen_manager' THEN ARRAY['cocina','productos','disponibilidad','recetas','inventario']::text[]
  WHEN 'warehouse' THEN ARRAY['inventario','kardex','compras']::text[]
  WHEN 'buyer' THEN ARRAY['inventario','kardex','compras','logistica']::text[]
  WHEN 'courier' THEN ARRAY['delivery']::text[]
  WHEN 'accounting' THEN ARRAY['dashboard','caja','costos','bi','fiscal','facturacion']::text[]
  WHEN 'auditor' THEN ARRAY['dashboard','productos','clientes','inventario','kardex','compras','costos','bi']::text[]
  ELSE ARRAY[]::text[]
END;
