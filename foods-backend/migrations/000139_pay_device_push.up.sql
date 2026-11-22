-- Fudia Pay: token de push (Firebase Cloud Messaging) por dispositivo para
-- despertar la app cuando el fabricante congela su proceso (HONOR/MagicOS).
-- El push no lleva datos del restaurante: solo pide procesar la bandeja.
-- woken_at limita los avisos a uno cada pocos segundos por dispositivo, aunque
-- varias pestañas o instancias de la API lo pidan a la vez.
ALTER TABLE pay_devices
  ADD COLUMN push_token text CHECK (push_token IS NULL OR length(push_token) BETWEEN 1 AND 4096),
  ADD COLUMN push_token_updated_at timestamptz,
  ADD COLUMN woken_at timestamptz;

CREATE INDEX pay_devices_push_location_idx ON pay_devices (organization_id, location_id)
  WHERE active AND push_token IS NOT NULL;
