-- Unifica el control de cantidad de productos vendidos.
-- El producto describe qué se vende; las cantidades viven por local.
ALTER TABLE products
  ADD COLUMN quantity_control text NOT NULL DEFAULT 'none'
    CHECK (quantity_control IN ('none','portions','inventory'));

UPDATE products
SET quantity_control = CASE stock_mode
  WHEN 'manual' THEN 'portions'
  WHEN 'linked' THEN 'inventory'
  ELSE 'none'
END;

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_quota_requires_manual;
ALTER TABLE products DROP COLUMN default_daily_quota;
ALTER TABLE products DROP COLUMN stock_mode;

ALTER TABLE product_availability
  RENAME COLUMN daily_quota TO portion_quantity;

COMMENT ON COLUMN products.quantity_control IS
  'none=sin cantidad; portions=porciones disponibles por local/día; inventory=existencia física desde Inventario.';
COMMENT ON COLUMN product_availability.portion_quantity IS
  'Cantidad de porciones preparadas disponibles para el local y día de negocio.';

-- Un producto físico vendible se vincula a un único registro interno de inventario.
-- inventory_items sigue pudiendo representar insumos no vendibles para recetas futuras.
ALTER TABLE inventory_items
  ADD COLUMN product_id uuid;

ALTER TABLE inventory_items
  ADD CONSTRAINT inventory_items_product_scope_fk
  FOREIGN KEY (product_id, organization_id)
  REFERENCES products(id, organization_id);

CREATE UNIQUE INDEX inventory_items_product_unique_idx
  ON inventory_items(organization_id, product_id)
  WHERE product_id IS NOT NULL;

ALTER TABLE stock_balances
  ADD CONSTRAINT stock_balances_non_negative CHECK (quantity >= 0);

CREATE TABLE inventory_entries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  product_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK (quantity > 0),
  unit text NOT NULL,
  note text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, organization_id),
  FOREIGN KEY (location_id, organization_id) REFERENCES locations(id, organization_id),
  FOREIGN KEY (product_id, organization_id) REFERENCES products(id, organization_id),
  FOREIGN KEY (inventory_item_id, organization_id) REFERENCES inventory_items(id, organization_id)
);

CREATE INDEX inventory_entries_scope_idx
  ON inventory_entries(organization_id, location_id, created_at DESC);

CREATE TABLE stock_movements (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  product_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  movement_type text NOT NULL
    CHECK (movement_type IN ('entry','sale','sale_reversal','sale_adjustment')),
  quantity_delta numeric(14,3) NOT NULL CHECK (quantity_delta <> 0),
  balance_after numeric(14,3) NOT NULL CHECK (balance_after >= 0),
  source_type text NOT NULL CHECK (source_type IN ('inventory_entry','order')),
  source_id uuid NOT NULL,
  note text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (location_id, organization_id) REFERENCES locations(id, organization_id),
  FOREIGN KEY (product_id, organization_id) REFERENCES products(id, organization_id),
  FOREIGN KEY (inventory_item_id, organization_id) REFERENCES inventory_items(id, organization_id)
);

CREATE INDEX stock_movements_product_idx
  ON stock_movements(organization_id, location_id, product_id, created_at DESC);
CREATE INDEX stock_movements_source_idx
  ON stock_movements(organization_id, source_type, source_id);
