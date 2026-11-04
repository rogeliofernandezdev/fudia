-- Pedidos de Fudia Concierge: quedan guardados (ítems «nuevo», fuera de Cocina)
-- hasta que el mozo o el cajero pulse «Enviar comanda». Cada ronda recuerda qué
-- WhatsApp la pidió para avisarle «en preparación» al enviarla.
ALTER TABLE concierge_order_requests
  ADD COLUMN customer_phone text NOT NULL DEFAULT '',
  ADD COLUMN round_id uuid;

ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_kind_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_kind_check
  CHECK (kind IN ('order_ready','order_sent'));
-- «Pedido listo» avisa una vez por pedido; «en preparación», una vez por ronda y WhatsApp.
ALTER TABLE customer_notices ADD COLUMN dedupe_key text NOT NULL DEFAULT ''
  CHECK (length(dedupe_key) <= 200);
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_order_id_kind_key;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_order_kind_dedupe_key
  UNIQUE (order_id,kind,dedupe_key);
