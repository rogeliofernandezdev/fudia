-- Pedidos sin cerrar del local: base del bloque «Operación ahora» del dashboard
-- (saldo pendiente, mesas ocupadas, cocina y delivery). Sin este índice parcial
-- la consulta recorre todos los pedidos históricos del local en cada carga.
CREATE INDEX IF NOT EXISTS orders_active_unsettled_idx
  ON orders(organization_id, location_id)
  WHERE status<>'cancelado' AND completed_at IS NULL;
