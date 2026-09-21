-- Completa Abastecimiento: mínimos por local, valorización, recepciones idempotentes,
-- devoluciones/correcciones de compra y transferencias entre locales.

-- Proveedores: consolidar RUC duplicados antes de exigir unicidad.
WITH ranked AS (
  SELECT id,organization_id,tax_id,
         first_value(id) OVER (PARTITION BY organization_id,tax_id ORDER BY id::text) AS canonical_id,
         row_number() OVER (PARTITION BY organization_id,tax_id ORDER BY id::text) AS rn
  FROM suppliers
  WHERE tax_id IS NOT NULL AND btrim(tax_id)<>''
)
UPDATE purchase_orders po
SET supplier_id=r.canonical_id
FROM ranked r
WHERE po.organization_id=r.organization_id AND po.supplier_id=r.id AND r.rn>1;

WITH ranked AS (
  SELECT id,organization_id,tax_id,
         row_number() OVER (PARTITION BY organization_id,tax_id ORDER BY id::text) AS rn
  FROM suppliers
  WHERE tax_id IS NOT NULL AND btrim(tax_id)<>''
)
DELETE FROM suppliers s
USING ranked r
WHERE s.id=r.id AND s.organization_id=r.organization_id AND r.rn>1;

CREATE UNIQUE INDEX IF NOT EXISTS suppliers_org_tax_id_uq
  ON suppliers(organization_id,tax_id)
  WHERE tax_id IS NOT NULL AND btrim(tax_id)<>'';

-- Configuración de inventario por sede.
CREATE TABLE inventory_location_settings (
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  minimum_stock numeric(14,3) NOT NULL DEFAULT 0 CHECK(minimum_stock>=0),
  reorder_point numeric(14,3) NOT NULL DEFAULT 0 CHECK(reorder_point>=0),
  optimal_stock numeric(14,3) NOT NULL DEFAULT 0 CHECK(optimal_stock>=0),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(location_id,inventory_item_id),
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY(inventory_item_id,organization_id) REFERENCES inventory_items(id,organization_id)
);

INSERT INTO inventory_location_settings(
  organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock
)
SELECT ii.organization_id,l.id,ii.id,ii.minimum_stock,ii.minimum_stock,ii.minimum_stock
FROM inventory_items ii
JOIN locations l ON l.organization_id=ii.organization_id
ON CONFLICT(location_id,inventory_item_id) DO NOTHING;

-- Costo promedio móvil por local.
ALTER TABLE stock_balances
  ADD COLUMN average_unit_cost numeric(14,4) NOT NULL DEFAULT 0 CHECK(average_unit_cost>=0);

ALTER TABLE stock_movements
  ADD COLUMN unit_cost numeric(14,4) NOT NULL DEFAULT 0 CHECK(unit_cost>=0),
  ADD COLUMN value_delta numeric(16,4) NOT NULL DEFAULT 0,
  ADD COLUMN balance_value_after numeric(16,4) NOT NULL DEFAULT 0 CHECK(balance_value_after>=0);

-- Idempotencia y costo histórico de recepciones.
ALTER TABLE purchase_receipts
  ADD COLUMN idempotency_key text;

CREATE UNIQUE INDEX purchase_receipts_idempotency_uq
  ON purchase_receipts(organization_id,location_id,idempotency_key)
  WHERE idempotency_key IS NOT NULL AND idempotency_key<>'';

ALTER TABLE purchase_receipt_items
  ADD COLUMN unit_cost numeric(14,4) NOT NULL DEFAULT 0 CHECK(unit_cost>=0);

UPDATE purchase_receipt_items pri
SET unit_cost=poi.unit_cost
FROM purchase_order_items poi
WHERE poi.id=pri.purchase_order_item_id AND poi.organization_id=pri.organization_id;

-- Las devoluciones referencian líneas por id + organización.
ALTER TABLE purchase_receipt_items
  ADD CONSTRAINT purchase_receipt_items_scope_uq UNIQUE(id,organization_id);

-- Devoluciones/correcciones contra una recepción.
CREATE TABLE purchase_returns (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  supplier_id uuid NOT NULL,
  purchase_order_id uuid NOT NULL,
  purchase_receipt_id uuid NOT NULL,
  code text NOT NULL DEFAULT ('DEV-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,8))),
  kind text NOT NULL CHECK(kind IN ('supplier_return','receipt_correction')),
  reason text NOT NULL,
  notes text NOT NULL DEFAULT '',
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  UNIQUE(organization_id,code),
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY(supplier_id,organization_id) REFERENCES suppliers(id,organization_id),
  FOREIGN KEY(purchase_order_id,organization_id) REFERENCES purchase_orders(id,organization_id),
  FOREIGN KEY(purchase_receipt_id,organization_id) REFERENCES purchase_receipts(id,organization_id)
);

CREATE TABLE purchase_return_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  purchase_return_id uuid NOT NULL,
  purchase_receipt_item_id uuid NOT NULL,
  purchase_order_item_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK(quantity>0),
  presentation_type text NOT NULL CHECK(presentation_type IN ('unit','package','box')),
  units_per_presentation numeric(14,3) NOT NULL CHECK(units_per_presentation>0),
  stock_quantity numeric(14,3) GENERATED ALWAYS AS (quantity*units_per_presentation) STORED,
  unit_cost numeric(14,4) NOT NULL DEFAULT 0 CHECK(unit_cost>=0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  FOREIGN KEY(purchase_return_id,organization_id) REFERENCES purchase_returns(id,organization_id) ON DELETE CASCADE,
  FOREIGN KEY(purchase_receipt_item_id,organization_id) REFERENCES purchase_receipt_items(id,organization_id),
  FOREIGN KEY(purchase_order_item_id,organization_id) REFERENCES purchase_order_items(id,organization_id),
  FOREIGN KEY(inventory_item_id,organization_id) REFERENCES inventory_items(id,organization_id)
);
CREATE INDEX purchase_returns_scope_idx ON purchase_returns(organization_id,location_id,created_at DESC);

-- Transferencias entre locales.
CREATE TABLE inventory_transfers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  from_location_id uuid NOT NULL,
  to_location_id uuid NOT NULL,
  code text NOT NULL DEFAULT ('TRF-'||upper(substr(replace(gen_random_uuid()::text,'-',''),1,8))),
  notes text NOT NULL DEFAULT '',
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  UNIQUE(organization_id,code),
  CHECK(from_location_id<>to_location_id),
  FOREIGN KEY(from_location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY(to_location_id,organization_id) REFERENCES locations(id,organization_id)
);

CREATE TABLE inventory_transfer_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  transfer_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK(quantity>0),
  unit_cost numeric(14,4) NOT NULL DEFAULT 0 CHECK(unit_cost>=0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  UNIQUE(transfer_id,inventory_item_id),
  FOREIGN KEY(transfer_id,organization_id) REFERENCES inventory_transfers(id,organization_id) ON DELETE CASCADE,
  FOREIGN KEY(inventory_item_id,organization_id) REFERENCES inventory_items(id,organization_id)
);
CREATE INDEX inventory_transfers_scope_idx ON inventory_transfers(organization_id,from_location_id,to_location_id,created_at DESC);

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check,
  ADD CONSTRAINT stock_movements_movement_type_check
    CHECK(movement_type IN (
      'entry','sale','sale_reversal','sale_adjustment','inventory_adjustment',
      'supplier_return','receipt_correction','transfer_out','transfer_in',
      'recipe_consumption','recipe_reversal'
    ));

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_source_type_check,
  ADD CONSTRAINT stock_movements_source_type_check
    CHECK(source_type IN (
      'inventory_entry','order','inventory_adjustment','purchase_receipt',
      'purchase_return','inventory_transfer'
    ));

-- Segregación de funciones sin romper roles existentes.
UPDATE roles
SET permissions=array_append(permissions,'purchases.approve')
WHERE permissions @> ARRAY['purchases.manage']::text[]
  AND NOT permissions @> ARRAY['purchases.approve']::text[];

UPDATE roles
SET permissions=array_append(permissions,'inventory.transfer')
WHERE permissions @> ARRAY['inventory.manage']::text[]
  AND NOT permissions @> ARRAY['inventory.transfer']::text[];
