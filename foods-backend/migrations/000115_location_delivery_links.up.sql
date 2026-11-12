-- Enlace y QR de delivery por local: abren el WhatsApp de Fudia ya ligado al
-- local para pedir delivery o recojo, sin haber escaneado antes el QR de una
-- mesa. El token se resuelve como el de un QR (mismo formato de 32 hex) con
-- serviceType=delivery; no es una mesa: no ocupa mesas ni cuenta en el plan.
CREATE TABLE location_delivery_links (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  qr_token text NOT NULL DEFAULT replace(gen_random_uuid()::text,'-','') CHECK (qr_token ~ '^[0-9a-f]{32}$'),
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (location_id),
  UNIQUE (qr_token),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);
