-- Nivel de control de inventario de la organización.
-- 'simple' opera sin insumos ni recetas; 'detailed' habilita inventario completo.
ALTER TABLE organizations
  ADD COLUMN inventory_mode text NOT NULL DEFAULT 'simple'
    CHECK (inventory_mode IN ('simple','detailed'));

-- Modo de control de disponibilidad del producto.
-- none: siempre disponible. manual: cupo del día escrito por el local.
-- linked: uno a uno con un insumo. recipe: ficha técnica de insumos.
ALTER TABLE products
  ADD COLUMN stock_mode text NOT NULL DEFAULT 'none'
    CHECK (stock_mode IN ('none','manual','linked','recipe')),
  ADD COLUMN default_daily_quota integer
    CHECK (default_daily_quota IS NULL OR default_daily_quota >= 0);

-- El cupo por defecto solo tiene sentido cuando el control es manual.
ALTER TABLE products
  ADD CONSTRAINT products_quota_requires_manual
    CHECK (stock_mode = 'manual' OR default_daily_quota IS NULL);
