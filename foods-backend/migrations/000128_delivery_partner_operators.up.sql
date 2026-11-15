-- Panel de la central de Delivery Fudia con inicio de sesión. El enlace privado
-- de la central (panel_token) deja de existir: cada operador entra con su correo
-- y contraseña. Plataforma lo da de alta con una contraseña temporal que el
-- operador cambia en su primer ingreso. El enlace del motorizado no cambia.

CREATE TABLE delivery_partner_operators (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id uuid NOT NULL REFERENCES delivery_partners(id) ON DELETE CASCADE,
  full_name text NOT NULL CHECK (length(btrim(full_name)) BETWEEN 2 AND 80),
  -- Correo en minúsculas; identifica al operador en el inicio de sesión.
  email text NOT NULL CHECK (email = lower(btrim(email)) AND length(email) <= 180 AND email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'),
  password_hash text NOT NULL,
  must_change_password boolean NOT NULL DEFAULT true,
  active boolean NOT NULL DEFAULT true,
  -- Bloqueo ante intentos fallidos seguidos.
  failed_logins integer NOT NULL DEFAULT 0 CHECK (failed_logins >= 0),
  locked_until timestamptz,
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX delivery_partner_operators_email_idx ON delivery_partner_operators(email);
CREATE INDEX delivery_partner_operators_partner_idx ON delivery_partner_operators(partner_id);

-- Sesiones del panel: solo se guarda el hash del token de la cookie.
CREATE TABLE delivery_partner_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  token_hash bytea NOT NULL UNIQUE,
  operator_id uuid NOT NULL REFERENCES delivery_partner_operators(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX delivery_partner_sessions_operator_idx ON delivery_partner_sessions(operator_id) WHERE revoked_at IS NULL;

-- Qué operador hizo cada acción del panel (aceptar, rechazar, liberar, asignar…).
CREATE TABLE delivery_partner_operator_actions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id uuid NOT NULL REFERENCES delivery_partners(id) ON DELETE CASCADE,
  operator_id uuid NOT NULL REFERENCES delivery_partner_operators(id) ON DELETE CASCADE,
  action text NOT NULL CHECK (length(action) BETWEEN 1 AND 40),
  outcome text NOT NULL DEFAULT '',
  order_code text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX delivery_partner_operator_actions_partner_idx ON delivery_partner_operator_actions(partner_id, created_at DESC);

DROP INDEX IF EXISTS delivery_partners_panel_token_idx;
ALTER TABLE delivery_partners DROP COLUMN panel_token;
