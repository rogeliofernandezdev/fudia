-- Separa qué clase de producto es de cómo se controla su cantidad.
-- Las categorías siguen siendo clasificación comercial compartida.
ALTER TABLE products
  ADD COLUMN product_type text NOT NULL DEFAULT 'prepared'
    CHECK (product_type IN ('prepared','retail'));

-- Hasta esta migración, Inventario era el único flujo que creaba productos
-- con quantity_control='inventory', por lo que esos registros son mercadería.
UPDATE products
SET product_type='retail'
WHERE quantity_control='inventory';

COMMENT ON COLUMN products.product_type IS
  'prepared=producto preparado por el restaurante; retail=mercadería vendible recibida físicamente. Independiente de categoría y quantity_control.';
