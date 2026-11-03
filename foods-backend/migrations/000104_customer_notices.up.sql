-- Bandeja de salida de avisos al cliente por WhatsApp. Cuando Cocina/Barra deja
-- listo todo un pedido de mostrador o recojo creado por Fudia Concierge, se
-- registra un aviso; Concierge lo reclama, lo encola en la conversación del
-- cliente y confirma el despacho. Un pedido avisa una sola vez por tipo.
CREATE TABLE customer_notices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  order_id uuid NOT NULL,
  kind text NOT NULL CHECK (kind IN ('order_ready')),
  customer_phone text NOT NULL CHECK (length(btrim(customer_phone)) BETWEEN 1 AND 32),
  conversation_id text NOT NULL CHECK (length(btrim(conversation_id)) BETWEEN 1 AND 160),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','dispatched')),
  attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  -- Reclamo con vencimiento: si Concierge no confirma, otro worker lo retoma.
  claimed_until timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  dispatched_at timestamptz,
  UNIQUE (order_id,kind),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id) ON DELETE CASCADE,
  CHECK ((status='pending' AND dispatched_at IS NULL) OR (status='dispatched' AND dispatched_at IS NOT NULL))
);

CREATE INDEX customer_notices_pending_idx
  ON customer_notices(created_at)
  WHERE status='pending';
