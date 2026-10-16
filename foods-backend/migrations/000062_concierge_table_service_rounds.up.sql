-- Separa la conversación de WhatsApp de cada solicitud idempotente y
-- modela rondas de cocina dentro de una única comanda/cuenta de mesa.

ALTER TABLE concierge_order_requests
  ADD COLUMN request_id text;

UPDATE concierge_order_requests
SET request_id=conversation_id
WHERE request_id IS NULL;

ALTER TABLE concierge_order_requests
  ALTER COLUMN request_id SET NOT NULL;

ALTER TABLE concierge_order_requests
  DROP CONSTRAINT concierge_order_requests_pkey;

ALTER TABLE concierge_order_requests
  ADD PRIMARY KEY (organization_id,location_id,request_id);

ALTER TABLE concierge_order_requests
  ADD CONSTRAINT concierge_order_requests_request_id_check
  CHECK (length(btrim(request_id)) BETWEEN 1 AND 160);

CREATE INDEX concierge_order_requests_conversation_idx
  ON concierge_order_requests(
    organization_id,
    location_id,
    conversation_id,
    created_at DESC
  );

CREATE TABLE order_kitchen_rounds (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  order_id uuid NOT NULL,
  sequence integer NOT NULL CHECK (sequence > 0),
  status text NOT NULL DEFAULT 'confirmado'
    CHECK (status IN ('confirmado','preparando','listo')),
  source text NOT NULL DEFAULT 'order'
    CHECK (source IN ('order','concierge')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id),
  UNIQUE (organization_id,location_id,order_id,sequence),
  FOREIGN KEY (location_id,organization_id)
    REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id)
    REFERENCES orders(id,organization_id) ON DELETE CASCADE
);

CREATE INDEX order_kitchen_rounds_queue_idx
  ON order_kitchen_rounds(
    organization_id,
    location_id,
    status,
    updated_at,
    created_at
  );

ALTER TABLE order_items
  ADD COLUMN kitchen_round_id uuid;

ALTER TABLE order_items
  ADD CONSTRAINT order_items_kitchen_round_fk
  FOREIGN KEY (kitchen_round_id,organization_id)
  REFERENCES order_kitchen_rounds(id,organization_id)
  ON DELETE SET NULL;

CREATE INDEX order_items_kitchen_round_idx
  ON order_items(organization_id,kitchen_round_id,created_at,id);
