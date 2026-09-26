-- Índices para rutas de lectura de alta frecuencia.
-- Evitan scans y mejoran sesión, salón, dashboard y listados administrativos.

CREATE INDEX IF NOT EXISTS user_roles_user_location_idx
  ON user_roles(user_id,location_id,role_id);

CREATE INDEX IF NOT EXISTS users_org_active_name_idx
  ON users(organization_id,active,full_name,id);

CREATE INDEX IF NOT EXISTS orders_open_location_idx
  ON orders(organization_id,location_id,created_at DESC)
  WHERE status NOT IN ('entregado','cancelado');

CREATE INDEX IF NOT EXISTS orders_open_table_lookup_idx
  ON orders(organization_id,location_id,table_id,created_at DESC)
  WHERE status NOT IN ('entregado','cancelado') AND table_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS payments_location_created_idx
  ON payments(organization_id,location_id,created_at DESC);

CREATE INDEX IF NOT EXISTS payment_refunds_location_created_idx
  ON payment_refunds(organization_id,location_id,created_at DESC);

CREATE INDEX IF NOT EXISTS reservations_active_schedule_idx
  ON reservations(organization_id,location_id,starts_at)
  WHERE status IN ('pending','confirmed');
