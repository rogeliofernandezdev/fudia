-- Una comanda de Concierge puede recibir varias rondas confirmadas.
-- Cada conversation_id sigue siendo idempotente, pero varias conversaciones/rondas
-- pueden referenciar la misma order_id mientras la comanda continúe editable.
ALTER TABLE concierge_order_requests
  DROP CONSTRAINT IF EXISTS concierge_order_requests_order_id_organization_id_key;

CREATE INDEX IF NOT EXISTS concierge_order_requests_order_idx
  ON concierge_order_requests(
    organization_id,
    location_id,
    order_id,
    created_at DESC
  );
