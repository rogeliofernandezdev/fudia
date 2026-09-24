DROP INDEX IF EXISTS concierge_order_requests_order_idx;

-- Para restaurar la restricción histórica se conserva la solicitud más antigua
-- por pedido. Esta reversión descarta únicamente las rondas adicionales.
WITH ranked AS (
  SELECT ctid,
         row_number() OVER (
           PARTITION BY order_id,organization_id
           ORDER BY created_at,conversation_id
         ) AS rn
  FROM concierge_order_requests
)
DELETE FROM concierge_order_requests c
USING ranked r
WHERE c.ctid=r.ctid AND r.rn>1;

ALTER TABLE concierge_order_requests
  ADD CONSTRAINT concierge_order_requests_order_id_organization_id_key
  UNIQUE(order_id,organization_id);
