-- Flujo auditable de órdenes de compra y sus líneas.
ALTER TABLE purchase_orders
  ADD CONSTRAINT purchase_orders_scope_uq UNIQUE (id, organization_id),
  ADD COLUMN notes text NOT NULL DEFAULT '',
  ADD COLUMN expected_at date,
  ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN approved_at timestamptz,
  ADD COLUMN received_at timestamptz,
  ADD COLUMN cancelled_at timestamptz;

CREATE TABLE purchase_order_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  purchase_order_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  presentation_id uuid NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK (quantity > 0),
  presentation_type text NOT NULL CHECK (presentation_type IN ('unit','package','box')),
  units_per_presentation numeric(14,3) NOT NULL CHECK (units_per_presentation > 0),
  stock_quantity numeric(14,3) GENERATED ALWAYS AS (quantity * units_per_presentation) STORED,
  unit_cost numeric(14,4) NOT NULL CHECK (unit_cost >= 0),
  line_total numeric(14,2) GENERATED ALWAYS AS (round(quantity * unit_cost, 2)) STORED,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, organization_id),
  UNIQUE (purchase_order_id, inventory_item_id, presentation_id),
  FOREIGN KEY (purchase_order_id, organization_id)
    REFERENCES purchase_orders(id, organization_id) ON DELETE CASCADE,
  FOREIGN KEY (inventory_item_id, organization_id)
    REFERENCES inventory_items(id, organization_id),
  FOREIGN KEY (presentation_id, organization_id)
    REFERENCES inventory_presentations(id, organization_id)
);

CREATE INDEX purchase_order_items_order_idx
  ON purchase_order_items(organization_id, purchase_order_id);

CREATE INDEX purchase_orders_status_idx
  ON purchase_orders(organization_id, location_id, status, created_at DESC);
