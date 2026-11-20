-- Índices para lecturas calientes que recorrían el historial completo.
-- Medidos con EXPLAIN (ANALYZE, BUFFERS) en PostgreSQL 16 desechable con un local
-- de un año (200 mil pedidos, 1,2 M movimientos de Kárdex, 1,4 M filas de
-- auditoría) y diez empresas de relleno; ver docs/05_DATABASE_GUIDELINES.md.
-- En producción con tablas grandes, crear cada índice con CONCURRENTLY fuera del
-- runner antes de desplegar: aquí usan IF NOT EXISTS.

-- Pedidos por local y fecha sin importar su estado: listado de Ventas/POS (orden
-- y rango de fechas) y cupos diarios de opciones de menú. Los índices existentes
-- intercalan estado, canal, mesa o una expresión antes de created_at.
CREATE INDEX IF NOT EXISTS orders_location_created_idx
  ON orders(organization_id,location_id,created_at DESC,id DESC);

-- Pedido QR/Concierge: «¿lo creó Concierge?» se consulta mientras el pedido está
-- bloqueado y recorría toda la auditoría (60 ms → 0,1 ms).
CREATE INDEX IF NOT EXISTS audit_log_concierge_order_idx
  ON audit_log(organization_id,entity_id)
  WHERE entity_type='order' AND action='concierge.order.created';

-- Kárdex: listado del local y de un artículo, en el orden de la pantalla
-- (110 ms y 43 ms → < 0,1 ms).
CREATE INDEX IF NOT EXISTS stock_movements_location_created_idx
  ON stock_movements(organization_id,location_id,created_at DESC,sequence_number DESC);
CREATE INDEX IF NOT EXISTS stock_movements_item_created_idx
  ON stock_movements(organization_id,location_id,inventory_item_id,created_at DESC,sequence_number DESC);

-- Barrido de delivery (cada 5-20 s): despachos que aún pueden cambiar.
CREATE INDEX IF NOT EXISTS delivery_dispatches_due_idx
  ON delivery_dispatches(order_id)
  WHERE status IN ('searching','accepted','unassigned');

-- Listado de clientes ordenado por última actualización.
CREATE INDEX IF NOT EXISTS customers_org_updated_idx
  ON customers(organization_id,updated_at DESC,display_name);
