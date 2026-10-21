DROP INDEX IF EXISTS order_items_kitchen_round_idx;

ALTER TABLE order_items
  DROP CONSTRAINT IF EXISTS order_items_kitchen_round_fk;

ALTER TABLE order_items
  DROP COLUMN IF EXISTS kitchen_round_id;

DROP TABLE IF EXISTS order_kitchen_rounds;

DROP INDEX IF EXISTS concierge_order_requests_conversation_idx;

ALTER TABLE concierge_order_requests
  DROP CONSTRAINT IF EXISTS concierge_order_requests_pkey;

WITH ranked AS (
  SELECT ctid,
         row_number() OVER (
           PARTITION BY organization_id,location_id,conversation_id
           ORDER BY created_at,request_id
         ) AS rn
  FROM concierge_order_requests
)
DELETE FROM concierge_order_requests c
USING ranked r
WHERE c.ctid=r.ctid AND r.rn>1;

ALTER TABLE concierge_order_requests
  ADD PRIMARY KEY (organization_id,location_id,conversation_id);

ALTER TABLE concierge_order_requests
  DROP CONSTRAINT IF EXISTS concierge_order_requests_request_id_check;

ALTER TABLE concierge_order_requests
  DROP COLUMN IF EXISTS request_id;
