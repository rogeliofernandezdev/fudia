-- Delivery y recojo por Fudia Concierge cuando la atención en mesa terminó.
-- Las tarifas son rangos por distancia en línea recta desde el local: el
-- primer rango cuyo límite cubre la distancia define el costo; más allá del
-- mayor límite no hay cobertura.
CREATE TABLE delivery_fee_tiers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  max_distance_km numeric(6,2) NOT NULL CHECK (max_distance_km > 0 AND max_distance_km <= 50),
  fee numeric(12,2) NOT NULL CHECK (fee >= 0 AND fee <= 999),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id,location_id,max_distance_km),
  FOREIGN KEY (location_id,organization_id)
    REFERENCES locations(id,organization_id) ON DELETE CASCADE
);

-- Idempotencia de pedidos delivery/recojo de Concierge: no ocupan mesa.
CREATE TABLE concierge_delivery_requests (
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  request_id text NOT NULL,
  conversation_id text NOT NULL,
  order_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id,location_id,request_id),
  FOREIGN KEY (location_id,organization_id)
    REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id)
    REFERENCES orders(id,organization_id) ON DELETE CASCADE,
  CHECK (length(btrim(request_id)) BETWEEN 1 AND 160),
  CHECK (length(btrim(conversation_id)) BETWEEN 1 AND 160)
);

CREATE INDEX concierge_delivery_requests_conversation_idx
  ON concierge_delivery_requests(organization_id,location_id,conversation_id,order_id);

-- Punto de entrega y distancia cotizada, para el despacho y la auditoría.
ALTER TABLE orders
  ADD COLUMN delivery_latitude numeric(9,6),
  ADD COLUMN delivery_longitude numeric(9,6),
  ADD COLUMN delivery_distance_km numeric(6,2),
  ADD CONSTRAINT orders_delivery_point_check CHECK (
    (delivery_latitude IS NULL) = (delivery_longitude IS NULL)
    AND (delivery_latitude IS NULL OR delivery_latitude BETWEEN -90 AND 90)
    AND (delivery_longitude IS NULL OR delivery_longitude BETWEEN -180 AND 180)
    AND (delivery_distance_km IS NULL OR delivery_distance_km >= 0)
  );
