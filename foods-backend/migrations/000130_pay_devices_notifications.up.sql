-- Fudia Pay: dispositivos Android del local que detectan las notificaciones de
-- pagos recibidos por Yape y Plin y las reportan a Caja para revisión y
-- confirmación manual. La vinculación usa un código temporal de un solo uso
-- mostrado como QR desde Caja; la credencial del dispositivo se guarda solo
-- como hash SHA-256 y se puede revocar. Una notificación detectada nunca marca
-- una venta como pagada por sí sola: solo queda registrada como evento.

CREATE TABLE pay_devices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  -- Nombre que reconoce el administrador, p. ej. «Android de caja 1».
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  -- Plataforma declarada por el contrato: ANDROID en esta etapa, IOS preparado.
  platform text NOT NULL CHECK (platform IN ('ANDROID','IOS')),
  model text NOT NULL DEFAULT '' CHECK (length(model) <= 80),
  -- Solo el hash SHA-256 del token del dispositivo; el token en claro se
  -- entrega una sola vez al activar la vinculación y nunca se guarda.
  token_hash bytea NOT NULL UNIQUE,
  token_suffix text NOT NULL DEFAULT '' CHECK (length(token_suffix) <= 8),
  active boolean NOT NULL DEFAULT true,
  revoked_at timestamptz,
  last_seen_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX pay_devices_location_idx ON pay_devices(location_id) WHERE active;
CREATE INDEX pay_devices_organization_idx ON pay_devices(organization_id);

-- Códigos temporales de vinculación: un solo uso, con expiración corta, y
-- nunca contienen credenciales permanentes.
CREATE TABLE pay_pairing_codes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  code text NOT NULL UNIQUE,
  created_by uuid NOT NULL REFERENCES users(id),
  expires_at timestamptz NOT NULL,
  used_at timestamptz,
  device_id uuid REFERENCES pay_devices(id),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX pay_pairing_codes_expiry_idx ON pay_pairing_codes(expires_at);

-- Notificaciones de pagos detectados. La unicidad (device_id, event_key) es la
-- idempotencia: el mismo evento reportado otra vez, incluso de forma
-- concurrente, no duplica el registro. La identidad estable del evento la
-- construye el dispositivo a partir de la notificación de origen.
CREATE TABLE pay_notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id uuid NOT NULL REFERENCES pay_devices(id) ON DELETE CASCADE,
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  provider text NOT NULL CHECK (provider IN ('yape','plin')),
  event_key text NOT NULL CHECK (length(event_key) BETWEEN 8 AND 200),
  sender_name text NOT NULL DEFAULT '' CHECK (length(sender_name) <= 80),
  amount numeric(12,2) NOT NULL CHECK (amount > 0),
  currency text NOT NULL CHECK (length(currency) BETWEEN 2 AND 8),
  occurred_at timestamptz NOT NULL,
  -- detected: recién llegada; associated: el cajero la ligó a un pedido;
  -- confirmed: el cobro real ya se registró con el flujo de pagos existente;
  -- dismissed: el cajero la descartó como irrelevante o duplicada.
  status text NOT NULL DEFAULT 'detected' CHECK (status IN ('detected','associated','confirmed','dismissed')),
  -- Pedido asociado por el cajero: mismo alcance de tenencia, forzado en la
  -- base como en el resto de pagos.
  order_id uuid,
  -- Pago real registrado por Caja: se valida en la aplicación con el módulo
  -- de pagos; sin clave foránea para no acoplar los módulos en la base.
  payment_id uuid,
  confirmed_by uuid,
  confirmed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (device_id, event_key),
  FOREIGN KEY (order_id, organization_id) REFERENCES orders(id, organization_id)
);
CREATE INDEX pay_notifications_location_recent_idx ON pay_notifications(location_id, created_at DESC);
CREATE INDEX pay_notifications_open_idx ON pay_notifications(location_id) WHERE status IN ('detected','associated');
