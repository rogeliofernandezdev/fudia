-- Recetas opcionales: no son requisito para Productos, Inventario, Compras ni Kárdex.
CREATE TABLE product_recipes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  product_id uuid NOT NULL,
  yield_quantity numeric(14,3) NOT NULL DEFAULT 1 CHECK(yield_quantity>0),
  notes text NOT NULL DEFAULT '',
  active boolean NOT NULL DEFAULT true,
  created_by uuid REFERENCES users(id),
  updated_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  UNIQUE(organization_id,product_id),
  FOREIGN KEY(product_id,organization_id) REFERENCES products(id,organization_id)
);

CREATE TABLE product_recipe_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  recipe_id uuid NOT NULL,
  inventory_item_id uuid NOT NULL,
  quantity numeric(14,3) NOT NULL CHECK(quantity>0),
  waste_percent numeric(7,4) NOT NULL DEFAULT 0 CHECK(waste_percent>=0 AND waste_percent<100),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  UNIQUE(recipe_id,inventory_item_id),
  FOREIGN KEY(recipe_id,organization_id) REFERENCES product_recipes(id,organization_id) ON DELETE CASCADE,
  FOREIGN KEY(inventory_item_id,organization_id) REFERENCES inventory_items(id,organization_id)
);

CREATE INDEX product_recipes_active_idx ON product_recipes(organization_id,active,product_id);
CREATE INDEX product_recipe_items_recipe_idx ON product_recipe_items(organization_id,recipe_id);
