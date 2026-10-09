-- Recepciones parciales sobre la orden de compra existente.
ALTER TABLE purchase_orders
  DROP CONSTRAINT IF EXISTS purchase_orders_status_check,
  ADD CONSTRAINT purchase_orders_status_check
    CHECK(status IN ('draft','pending_approval','approved','partially_received','received','cancelled'));

ALTER TABLE purchase_order_items
  ADD COLUMN received_quantity numeric(14,3) NOT NULL DEFAULT 0
    CHECK (received_quantity >= 0 AND received_quantity <= quantity);

CREATE TABLE purchase_receipts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  purchase_order_id uuid NOT NULL,
  code text NOT NULL DEFAULT ('REC-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 8))),
  notes text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, organization_id),
  UNIQUE (organization_id, code),
  FOREIGN KEY (location_id, organization_id)
    REFERENCES locations(id, organization_id),
  FOREIGN KEY (purchase_order_id, organization_id)
    REFERENCES purchase_orders(id, organization_id)
);

CREATE TABLE purchase_receipt_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  purchase_receipt_id uuid NOT NULL,
  purchase_order_item_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  presentation_id uuid NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK (quantity > 0),
  presentation_type text NOT NULL CHECK (presentation_type IN ('unit','package','box')),
  units_per_presentation numeric(14,3) NOT NULL CHECK (units_per_presentation > 0),
  stock_quantity numeric(14,3) GENERATED ALWAYS AS (quantity * units_per_presentation) STORED,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (purchase_receipt_id, purchase_order_item_id),
  FOREIGN KEY (purchase_receipt_id, organization_id)
    REFERENCES purchase_receipts(id, organization_id) ON DELETE CASCADE,
  FOREIGN KEY (purchase_order_item_id, organization_id)
    REFERENCES purchase_order_items(id, organization_id),
  FOREIGN KEY (inventory_item_id, organization_id)
    REFERENCES inventory_items(id, organization_id),
  FOREIGN KEY (presentation_id, organization_id)
    REFERENCES inventory_presentations(id, organization_id)
);

CREATE INDEX purchase_receipts_order_idx
  ON purchase_receipts(organization_id, purchase_order_id, created_at DESC);

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_source_type_check,
  ADD CONSTRAINT stock_movements_source_type_check
    CHECK (source_type IN ('inventory_entry','order','inventory_adjustment','purchase_receipt'));
