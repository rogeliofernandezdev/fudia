CREATE TABLE subscription_plans (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  currency char(3) NOT NULL,
  monthly_price numeric(12,2) NOT NULL CHECK(monthly_price>=0),
  annual_price numeric(12,2) NOT NULL CHECK(annual_price>=0),
  trial_days integer NOT NULL DEFAULT 0 CHECK(trial_days BETWEEN 0 AND 365),
  max_locations integer CHECK(max_locations IS NULL OR max_locations>0),
  max_users integer CHECK(max_users IS NULL OR max_users>0),
  module_keys text[] NOT NULL DEFAULT '{}',
  terms_version text NOT NULL,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK(code ~ '^[a-z0-9][a-z0-9_-]{1,39}$')
);

CREATE TABLE organization_subscriptions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL UNIQUE REFERENCES organizations(id),
  plan_id uuid NOT NULL REFERENCES subscription_plans(id),
  billing_cycle text NOT NULL CHECK(billing_cycle IN ('monthly','annual')),
  price_amount numeric(12,2) NOT NULL CHECK(price_amount>=0),
  currency char(3) NOT NULL,
  status text NOT NULL CHECK(status IN ('trial','active','past_due','cancelled')),
  trial_starts_at timestamptz,
  trial_ends_at timestamptz,
  current_period_starts_at timestamptz,
  current_period_ends_at timestamptz,
  renews_at timestamptz,
  auto_renew boolean NOT NULL DEFAULT true,
  terms_version text,
  terms_accepted_at timestamptz,
  terms_accepted_by uuid REFERENCES users(id),
  started_at timestamptz NOT NULL DEFAULT now(),
  cancelled_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscription_payments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  subscription_id uuid NOT NULL REFERENCES organization_subscriptions(id) ON DELETE CASCADE,
  amount numeric(12,2) NOT NULL CHECK(amount>=0),
  currency char(3) NOT NULL,
  status text NOT NULL CHECK(status IN ('pending','paid','failed','refunded')),
  provider text NOT NULL DEFAULT 'manual',
  external_reference text,
  period_starts_at timestamptz,
  period_ends_at timestamptz,
  paid_at timestamptz,
  recorded_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(provider,external_reference)
);

CREATE INDEX organization_subscriptions_status_idx ON organization_subscriptions(status,renews_at);
CREATE INDEX subscription_payments_org_idx ON subscription_payments(organization_id,created_at DESC);

INSERT INTO subscription_plans(
  code,name,description,currency,monthly_price,annual_price,trial_days,
  max_locations,max_users,module_keys,terms_version,active
) VALUES (
  'legacy','Plan legado','Compatibilidad para empresas creadas antes del catálogo comercial de planes.',
  'PEN',0,0,0,NULL,NULL,
  ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','compras','clientes','locales','fiscal','usuarios']::text[],
  'legacy',false
)
ON CONFLICT(code) DO NOTHING;

INSERT INTO organization_subscriptions(
  organization_id,plan_id,billing_cycle,price_amount,currency,status,auto_renew
)
SELECT o.id,p.id,'monthly',0,COALESCE(fp.currency,'PEN'),'active',false
FROM organizations o
JOIN subscription_plans p ON p.code='legacy'
LEFT JOIN LATERAL (
  SELECT currency
  FROM organization_fiscal_profiles
  WHERE organization_id=o.id AND is_default
  ORDER BY active DESC,created_at
  LIMIT 1
) fp ON true
ON CONFLICT(organization_id) DO NOTHING;
