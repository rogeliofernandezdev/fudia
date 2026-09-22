CREATE TABLE reservations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  location_id uuid NOT NULL,
  customer_id uuid,
  customer_name text NOT NULL,
  customer_phone text NOT NULL DEFAULT '',
  starts_at timestamptz NOT NULL,
  guests integer NOT NULL CHECK (guests > 0 AND guests <= 100),
  table_id uuid,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','seated','cancelled','no_show')),
  notes text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (customer_id,organization_id) REFERENCES customers(id,organization_id),
  FOREIGN KEY (table_id,organization_id,location_id) REFERENCES tables(id,organization_id,location_id)
);

CREATE INDEX reservations_schedule_idx
  ON reservations(organization_id,location_id,starts_at,status);

INSERT INTO organization_modules(organization_id,module_key,active)
SELECT o.id,m.key,m.active
FROM organizations o
CROSS JOIN (VALUES
  ('reportes',true),('pos',true),('pedidos',true),('cocina',true),('mesas',true),('caja',true),
  ('reservas',true),('productos',true),('combos',true),('recetas',true),('inventario',true),
  ('kardex',true),('compras',true),('clientes',true),('locales',true),('fiscal',true),('usuarios',true),
  ('call_center',false),('carta_qr',false),('kiosco',false),('logistica',false),('delivery',false),
  ('delivery_apps',false),('repartidores',false),('crm',false),('puntos',false),('ofertas',false),
  ('personal',false),('costos',false),('bi',false),('app_manager',false),('facturacion',false),
  ('integraciones',false),('whatsapp_bot',false)
) AS m(key,active)
ON CONFLICT (organization_id,module_key)
DO UPDATE SET active=EXCLUDED.active,updated_at=now();

UPDATE roles
SET menu_access=CASE system_key
  WHEN 'location_manager' THEN ARRAY['dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','recetas','inventario','kardex','compras','clientes','locales','fiscal','usuarios']::text[]
  WHEN 'shift_supervisor' THEN ARRAY['dashboard','pos','pedidos','cocina','mesas','caja','reservas','productos','disponibilidad','combos','clientes']::text[]
  WHEN 'cashier' THEN ARRAY['pos','pedidos','mesas','caja','clientes']::text[]
  WHEN 'waiter' THEN ARRAY['pos','pedidos','mesas','reservas']::text[]
  WHEN 'buyer' THEN ARRAY['inventario','kardex','compras']::text[]
  WHEN 'accounting' THEN ARRAY['dashboard','caja','fiscal']::text[]
  ELSE menu_access END,
permissions=CASE system_key
  WHEN 'location_manager' THEN ARRAY['dashboard.read','organizations.read','organizations.manage','users.read','users.manage','menu.read','menu.manage','recipes.manage','customers.read','customers.manage','orders.read','orders.manage','kitchen.manage','tables.read','tables.manage','cash.read','cash.manage','cash.expected.read','reservations.read','reservations.manage','inventory.read','inventory.manage','inventory.transfer','purchases.read','purchases.manage','purchases.approve','purchases.receive','reports.read']::text[]
  WHEN 'shift_supervisor' THEN ARRAY['dashboard.read','menu.read','customers.read','customers.manage','orders.read','orders.manage','kitchen.manage','tables.read','tables.manage','cash.read','cash.manage','cash.expected.read','reservations.read','reservations.manage']::text[]
  WHEN 'cashier' THEN ARRAY['menu.read','customers.read','customers.manage','orders.read','orders.manage','tables.read','cash.read','cash.manage']::text[]
  WHEN 'waiter' THEN ARRAY['menu.read','customers.read','customers.manage','orders.read','orders.manage','tables.read','reservations.read','reservations.manage']::text[]
  WHEN 'buyer' THEN ARRAY['inventory.read','purchases.read','purchases.manage','purchases.approve','purchases.receive']::text[]
  WHEN 'accounting' THEN ARRAY['dashboard.read','cash.read','cash.expected.read','reports.read','fiscal.read','organizations.read']::text[]
  ELSE permissions END,
updated_at=now()
WHERE system_key IN ('location_manager','shift_supervisor','cashier','waiter','buyer','accounting');
