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
