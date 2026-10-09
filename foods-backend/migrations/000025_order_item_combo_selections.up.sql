ALTER TABLE order_items
  ADD COLUMN item_type text NOT NULL DEFAULT 'product'
  CHECK (item_type IN ('product','combo'));

CREATE TABLE order_item_combo_selections (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  order_item_id uuid NOT NULL,
  group_id uuid NOT NULL,
  group_name text NOT NULL,
  option_product_id uuid NOT NULL,
  option_name text NOT NULL,
  surcharge numeric(12,2) NOT NULL DEFAULT 0 CHECK (surcharge >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (order_item_id, organization_id)
    REFERENCES order_items(id, organization_id) ON DELETE CASCADE,
  UNIQUE (order_item_id, group_id, option_product_id)
);

CREATE INDEX order_item_combo_selections_item_idx
  ON order_item_combo_selections(order_item_id);
