-- Ajustes manuales de inventario con trazabilidad completa.
CREATE TABLE inventory_adjustments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  movement_type text NOT NULL CHECK (movement_type IN ('entry','exit')),
  reason text NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK (quantity > 0),
  stock_before numeric(14,3) NOT NULL CHECK (stock_before >= 0),
  stock_after numeric(14,3) NOT NULL CHECK (stock_after >= 0),
  observation text NOT NULL DEFAULT '',
  created_by uuid NOT NULL REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, organization_id),
  FOREIGN KEY (location_id, organization_id)
    REFERENCES locations(id, organization_id),
  FOREIGN KEY (inventory_item_id, organization_id)
    REFERENCES inventory_items(id, organization_id),
  CHECK (
    (movement_type='entry' AND reason='surplus_adjustment')
    OR
    (movement_type='exit' AND reason IN ('shortage_adjustment','waste','expiration','other_exit'))
  )
);

CREATE INDEX inventory_adjustments_scope_idx
  ON inventory_adjustments(organization_id, location_id, inventory_item_id, created_at DESC);

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_movement_type_check,
  ADD CONSTRAINT stock_movements_movement_type_check
    CHECK (movement_type IN ('entry','sale','sale_reversal','sale_adjustment','inventory_adjustment'));

ALTER TABLE stock_movements
  DROP CONSTRAINT IF EXISTS stock_movements_source_type_check,
  ADD CONSTRAINT stock_movements_source_type_check
    CHECK (source_type IN ('inventory_entry','order','inventory_adjustment'));
