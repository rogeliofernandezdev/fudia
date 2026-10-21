-- Delivery to a table and final payment/closure are separate events.
ALTER TABLE orders ADD COLUMN completed_at timestamptz;
UPDATE orders SET completed_at=updated_at WHERE status='entregado';

DROP INDEX orders_one_open_table_uq;
DROP INDEX orders_open_location_idx;
DROP INDEX orders_open_table_lookup_idx;

CREATE UNIQUE INDEX orders_one_open_table_uq
  ON orders(organization_id,location_id,table_id)
  WHERE table_id IS NOT NULL AND status<>'cancelado'
    AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL));
CREATE INDEX orders_open_location_idx
  ON orders(organization_id,location_id,created_at DESC)
  WHERE status<>'cancelado'
    AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL));
CREATE INDEX orders_open_table_lookup_idx
  ON orders(organization_id,location_id,table_id,created_at DESC)
  WHERE table_id IS NOT NULL AND status<>'cancelado'
    AND (status<>'entregado' OR (channel='salon' AND completed_at IS NULL));
