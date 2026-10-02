-- Define para qué tipo de producto puede usarse cada categoría comercial.
ALTER TABLE menu_categories
  ADD COLUMN product_scope text NOT NULL DEFAULT 'prepared'
    CHECK (product_scope IN ('prepared','retail','both'));

-- Inferencia basada en los productos ya asociados, nunca en el nombre de la categoría.
UPDATE menu_categories c
SET product_scope = CASE
  WHEN EXISTS (
    SELECT 1 FROM products p
    WHERE p.organization_id=c.organization_id
      AND p.category_id=c.id
      AND p.product_type='retail'
  ) AND EXISTS (
    SELECT 1 FROM products p
    WHERE p.organization_id=c.organization_id
      AND p.category_id=c.id
      AND p.product_type='prepared'
  ) THEN 'both'
  WHEN EXISTS (
    SELECT 1 FROM products p
    WHERE p.organization_id=c.organization_id
      AND p.category_id=c.id
      AND p.product_type='retail'
  ) THEN 'retail'
  ELSE 'prepared'
END;

COMMENT ON COLUMN menu_categories.product_scope IS
  'prepared=solo platos/productos preparados; retail=solo mercadería vendible; both=ambos tipos.';
