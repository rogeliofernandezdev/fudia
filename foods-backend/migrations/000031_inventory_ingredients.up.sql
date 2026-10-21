-- Permite inventariar insumos internos que no son productos vendibles.
-- Carne, papa, aceite, etc. viven en inventory_items con product_id NULL.
ALTER TABLE inventory_items
  ADD COLUMN created_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

ALTER TABLE inventory_entries
  ALTER COLUMN product_id DROP NOT NULL;

ALTER TABLE stock_movements
  ALTER COLUMN product_id DROP NOT NULL;

COMMENT ON COLUMN inventory_items.product_id IS
  'Producto vendible vinculado. NULL para insumos internos usados por compras/recetas.';

COMMENT ON COLUMN inventory_entries.product_id IS
  'Producto vendible relacionado cuando aplica; NULL para entradas de insumos.';

COMMENT ON COLUMN stock_movements.product_id IS
  'Producto vendible relacionado cuando aplica; NULL para movimientos de insumos.';
