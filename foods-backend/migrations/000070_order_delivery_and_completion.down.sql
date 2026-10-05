-- Preserve open table balances when reverting to the former delivery model.
UPDATE orders SET status='listo'
  WHERE channel='salon' AND status='entregado' AND completed_at IS NULL;

DROP INDEX orders_one_open_table_uq;
DROP INDEX orders_open_location_idx;
DROP INDEX orders_open_table_lookup_idx;

CREATE UNIQUE INDEX orders_one_open_table_uq
  ON orders(organization_id,location_id,table_id)
  WHERE table_id IS NOT NULL AND status NOT IN ('entregado','cancelado');
CREATE INDEX orders_open_location_idx
  ON orders(organization_id,location_id,created_at DESC)
  WHERE status NOT IN ('entregado','cancelado');
CREATE INDEX orders_open_table_lookup_idx
  ON orders(organization_id,location_id,table_id,created_at DESC)
  WHERE table_id IS NOT NULL AND status NOT IN ('entregado','cancelado');

ALTER TABLE orders DROP COLUMN completed_at;
