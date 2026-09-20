CREATE TABLE orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  location_id uuid NOT NULL REFERENCES locations(id),
  code text NOT NULL DEFAULT ('PED-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 8))),
  channel text NOT NULL CHECK (channel IN ('salon','mostrador','recojo','delivery','whatsapp')),
  status text NOT NULL DEFAULT 'nuevo' CHECK (status IN ('nuevo','confirmado','preparando','listo','en_camino','entregado','cancelado')),
  customer_id uuid,
  customer_name text NOT NULL DEFAULT '',
  customer_phone text NOT NULL DEFAULT '',
  address text NOT NULL DEFAULT '',
  reference text NOT NULL DEFAULT '',
  table_id uuid,
  notes text NOT NULL DEFAULT '',
  subtotal numeric(12,2) NOT NULL DEFAULT 0 CHECK (subtotal >= 0),
  delivery_fee numeric(12,2) NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),
  total numeric(12,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, code),
  UNIQUE (id, organization_id),
  FOREIGN KEY (customer_id, organization_id) REFERENCES customers(id, organization_id),
  FOREIGN KEY (table_id, organization_id) REFERENCES tables(id, organization_id)
);
CREATE INDEX orders_scope_idx ON orders(organization_id, location_id, status, created_at DESC);
CREATE INDEX orders_channel_idx ON orders(organization_id, location_id, channel, created_at DESC);

CREATE TABLE order_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  order_id uuid NOT NULL,
  product_id uuid,
  name text NOT NULL,
  qty numeric(10,2) NOT NULL CHECK (qty > 0),
  unit_price numeric(12,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
  note text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, organization_id),
  FOREIGN KEY (order_id, organization_id) REFERENCES orders(id, organization_id) ON DELETE CASCADE,
  FOREIGN KEY (product_id, organization_id) REFERENCES products(id, organization_id)
);
CREATE INDEX order_items_order_idx ON order_items(order_id);
