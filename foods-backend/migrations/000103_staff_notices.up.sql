-- Avisos al personal (campana del panel): un pedido por QR de mesa o de
-- mostrador, un pedido de delivery o recojo, la cuenta pedida, un llamado al
-- mozo o una comanda lista. La audiencia se resuelve al leer por permisos del
-- usuario en el local: 'service' (tables.read), 'cashier' (cash.manage) y 'all'.
CREATE TABLE staff_notices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  kind text NOT NULL CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested')),
  audience text NOT NULL CHECK (audience IN ('service','cashier','all')),
  title text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 120),
  detail text NOT NULL DEFAULT '' CHECK (length(detail) <= 240),
  order_id uuid,
  table_id uuid,
  -- Reintentos del mismo evento (requestId de Concierge, comanda lista) no duplican el aviso.
  dedupe_key text CHECK (dedupe_key IS NULL OR length(btrim(dedupe_key)) BETWEEN 1 AND 200),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX staff_notices_dedupe_uq
  ON staff_notices(organization_id,dedupe_key)
  WHERE dedupe_key IS NOT NULL;
CREATE INDEX staff_notices_feed_idx
  ON staff_notices(organization_id,location_id,created_at DESC,id DESC);

-- Lectura por usuario: un aviso compartido por varios mozos queda pendiente
-- para cada uno hasta que lo abra.
CREATE TABLE staff_notice_reads (
  notice_id uuid NOT NULL REFERENCES staff_notices(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  read_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (notice_id,user_id)
);
