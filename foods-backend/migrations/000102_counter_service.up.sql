-- Venta por mostrador: el tipo de atención vive en la tabla actual de mesas.
-- Toda mesa existente queda como mesa tradicional ('table').
ALTER TABLE tables
  ADD COLUMN service_type text NOT NULL DEFAULT 'table',
  ADD CONSTRAINT tables_service_type_check CHECK (service_type IN ('table','counter'));

-- Consentimiento promocional por WhatsApp. customers.marketing_consent sigue
-- siendo el valor vigente; esta tabla guarda cada pregunta y respuesta con su
-- fecha y evidencia (texto literal del cliente o autor en Clientes).
CREATE TABLE customer_marketing_consents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  customer_id uuid NOT NULL,
  decision text NOT NULL CHECK (decision IN ('asked','granted','declined','withdrawn')),
  source text NOT NULL CHECK (source IN ('whatsapp','admin')),
  evidence text NOT NULL DEFAULT '' CHECK (length(evidence) <= 500),
  conversation_id text NOT NULL DEFAULT '' CHECK (length(conversation_id) <= 160),
  request_id text CHECK (request_id IS NULL OR length(btrim(request_id)) BETWEEN 1 AND 160),
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (customer_id,organization_id) REFERENCES customers(id,organization_id) ON DELETE CASCADE,
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id)
);

-- Reintentos de Concierge con el mismo requestId no duplican la evidencia.
CREATE UNIQUE INDEX customer_marketing_consents_request_uq
  ON customer_marketing_consents(organization_id,request_id)
  WHERE request_id IS NOT NULL;
CREATE INDEX customer_marketing_consents_customer_idx
  ON customer_marketing_consents(organization_id,customer_id,created_at DESC,id DESC);
