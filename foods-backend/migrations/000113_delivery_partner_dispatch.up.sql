-- Delivery Fudia: centrales de reparto con las que trabaja la plataforma.
-- Plataforma las registra; cada local elige su modo y hasta tres centrales
-- en orden de prioridad. Al confirmar un pedido de delivery se ofrece a la
-- 1.ª central por WhatsApp (Aceptar / Rechazar); si rechaza o no responde en
-- 3 minutos pasa a la siguiente. Concierge envía los mensajes desde la bandeja
-- de salida delivery_partner_messages; el backend nunca llama a Meta.
CREATE TABLE delivery_partners (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 2 AND 120),
  -- Número de WhatsApp con código de país, solo dígitos (E.164 sin «+»).
  whatsapp_phone text NOT NULL CHECK (whatsapp_phone ~ '^[1-9][0-9]{7,14}$'),
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (whatsapp_phone)
);

-- Modo de delivery del local: 'own' (reparto propio, Fudia solo recibe el
-- pedido) o 'fudia' (centrales de la plataforma). Sin fila equivale a 'own'.
CREATE TABLE location_delivery_settings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  mode text NOT NULL DEFAULT 'own' CHECK (mode IN ('own','fudia')),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (location_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);

-- Centrales del local en orden: 1.ª, 2.ª y 3.ª.
CREATE TABLE location_delivery_partners (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  partner_id uuid NOT NULL REFERENCES delivery_partners(id),
  priority smallint NOT NULL CHECK (priority BETWEEN 1 AND 3),
  UNIQUE (location_id,priority),
  UNIQUE (location_id,partner_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);

-- Despacho de un pedido de delivery a las centrales. Un pedido tiene a lo sumo
-- un despacho; «Reintentar» abre una nueva vuelta (round) desde la 1.ª central.
--   searching   ofrecido a una central, esperando respuesta
--   accepted    una central aceptó; recibe «listo para recoger» al terminar cocina
--   unassigned  ninguna central aceptó; la campana avisa a caja
--   picked_up   la central recogió el pedido (el pedido pasa a «en camino»)
--   own         el local decidió usar su delivery propio
--   cancelled   el pedido se canceló
CREATE TABLE delivery_dispatches (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  order_id uuid NOT NULL,
  status text NOT NULL DEFAULT 'searching'
    CHECK (status IN ('searching','accepted','unassigned','picked_up','own','cancelled')),
  round smallint NOT NULL DEFAULT 1 CHECK (round >= 1),
  accepted_partner_id uuid REFERENCES delivery_partners(id),
  accepted_at timestamptz,
  ready_notified_at timestamptz,
  picked_up_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (order_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id) ON DELETE CASCADE,
  CHECK (status NOT IN ('accepted','picked_up') OR accepted_partner_id IS NOT NULL)
);

CREATE INDEX delivery_dispatches_open_idx
  ON delivery_dispatches(status,updated_at)
  WHERE status IN ('searching','accepted');

-- Cada oferta a una central. action_token viaja en los botones de WhatsApp
-- (Aceptar, Rechazar y luego Recoger); lo genera PostgreSQL.
CREATE TABLE delivery_dispatch_offers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  dispatch_id uuid NOT NULL REFERENCES delivery_dispatches(id) ON DELETE CASCADE,
  partner_id uuid NOT NULL REFERENCES delivery_partners(id),
  round smallint NOT NULL CHECK (round >= 1),
  priority smallint NOT NULL CHECK (priority BETWEEN 1 AND 3),
  status text NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','accepted','rejected','expired','withdrawn')),
  action_token uuid NOT NULL DEFAULT gen_random_uuid(),
  -- Vence 3 minutos después de que Concierge confirma el envío del mensaje.
  expires_at timestamptz,
  responded_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (action_token),
  UNIQUE (dispatch_id,round,priority)
);

CREATE INDEX delivery_dispatch_offers_pending_idx
  ON delivery_dispatch_offers(dispatch_id)
  WHERE status='pending';

-- Bandeja de salida hacia las centrales: oferta (Aceptar/Rechazar), listo para
-- recoger (Recoger) y pedido cancelado. Concierge la reclama como customer_notices.
CREATE TABLE delivery_partner_messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  dispatch_id uuid NOT NULL REFERENCES delivery_dispatches(id) ON DELETE CASCADE,
  offer_id uuid NOT NULL REFERENCES delivery_dispatch_offers(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('offer','ready','cancelled')),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','dispatched')),
  attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  claimed_until timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  dispatched_at timestamptz,
  UNIQUE (offer_id,kind),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  CHECK ((status='pending' AND dispatched_at IS NULL) OR (status='dispatched' AND dispatched_at IS NOT NULL))
);

CREATE INDEX delivery_partner_messages_pending_idx
  ON delivery_partner_messages(created_at)
  WHERE status='pending';

-- Campana: ninguna central aceptó el delivery.
ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account','delivery_unassigned'));
