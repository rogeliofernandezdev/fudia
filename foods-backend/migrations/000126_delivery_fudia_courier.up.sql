-- Delivery Fudia: motorizado con enlace propio, liberación del pedido por la
-- central, alertas de recogida y entrega fallida. No agrega estados: el despacho
-- sigue en searching/accepted/picked_up/delivered/cancelled y el pedido en sus
-- estados actuales; estas columnas solo registran hechos del reparto.

-- Motorizado responsable. La central escribe su nombre antes de obtener el
-- enlace; PostgreSQL genera courier_token y lo reemplaza cuando cambia el nombre.
-- pickup_alert_ack_level: nivel de alerta de recogida que la central decidió
-- mantener (1 = 3 min, 2 = 5 min, 3 = 8 min). pickup_alerted_at: aviso crítico al
-- restaurante ya enviado. delivery_issue_*: el motorizado marcó «No pude entregar».
ALTER TABLE delivery_dispatches
  ADD COLUMN courier_name text CHECK (courier_name IS NULL OR char_length(courier_name) BETWEEN 2 AND 60),
  ADD COLUMN courier_token text CHECK (courier_token IS NULL OR courier_token ~ '^[a-f0-9]{32}$'),
  ADD COLUMN courier_assigned_at timestamptz,
  ADD COLUMN pickup_alert_ack_level smallint NOT NULL DEFAULT 0 CHECK (pickup_alert_ack_level BETWEEN 0 AND 3),
  ADD COLUMN pickup_alerted_at timestamptz,
  ADD COLUMN delivery_issue_reported_at timestamptz,
  ADD COLUMN delivery_issue_distance_m integer CHECK (delivery_issue_distance_m IS NULL OR delivery_issue_distance_m >= 0),
  ADD CONSTRAINT delivery_dispatches_courier_check CHECK (
    (courier_name IS NULL) = (courier_token IS NULL) AND (courier_name IS NULL) = (courier_assigned_at IS NULL)
  );
CREATE UNIQUE INDEX delivery_dispatches_courier_token_idx ON delivery_dispatches(courier_token) WHERE courier_token IS NOT NULL;

-- La central que libera un pedido no vuelve a recibirlo automáticamente.
ALTER TABLE delivery_dispatch_offers ADD COLUMN released_at timestamptz;

-- Entrega fallida registrada por la central: conserva los datos y montos para
-- gestionar el reclamo. La central reembolsa al motorizado el 100 % de la comida;
-- la pérdida se reparte entre Fudia y la central según la validación y los límites.
CREATE TABLE delivery_failed_deliveries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  dispatch_id uuid NOT NULL UNIQUE REFERENCES delivery_dispatches(id) ON DELETE CASCADE,
  order_id uuid NOT NULL,
  partner_id uuid NOT NULL REFERENCES delivery_partners(id),
  courier_name text NOT NULL,
  customer_name text NOT NULL,
  customer_phone text NOT NULL,
  address text NOT NULL,
  food_amount numeric(12,2) NOT NULL CHECK (food_amount >= 0),
  courier_reimbursement numeric(12,2) NOT NULL,
  partner_share numeric(12,2) NOT NULL CHECK (partner_share >= 0),
  fudia_share numeric(12,2) NOT NULL CHECK (fudia_share >= 0),
  fudia_share_eligible boolean NOT NULL,
  validation_notes text[] NOT NULL DEFAULT '{}',
  courier_distance_m integer,
  customer_notified boolean NOT NULL,
  reported_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id) ON DELETE CASCADE,
  CHECK (courier_reimbursement = food_amount),
  CHECK (partner_share + fudia_share = food_amount),
  CHECK (fudia_share_eligible OR fudia_share = 0)
);
CREATE INDEX delivery_failed_deliveries_partner_idx ON delivery_failed_deliveries(partner_id, created_at);

-- Campana: el motorizado no llega a recoger un pedido listo (alerta crítica, 8 min).
ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account','delivery_unassigned','delivery_pickup_delayed'));

-- Chat del cliente: el motorizado llegó y no lo ubica; se le pide comunicarse.
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_kind_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_kind_check
  CHECK (kind IN ('order_ready','order_sent','order_on_the_way','delivery_contact_request'));
