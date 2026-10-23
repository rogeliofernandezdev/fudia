-- Monthly Fudia Concierge conversation limit. Everything here is configured and
-- read only by the platform administrator; companies never see caps or usage.
-- A conversation is one complete attention (QR start, or delivery/pickup after
-- the table) and is counted once per company and month in the company timezone.

-- Per-plan cap; NULL is unlimited. Initial base caps (2026-10-23), still to be
-- confirmed with one month of measured cost.
ALTER TABLE subscription_plans
  ADD COLUMN max_concierge_conversations integer
  CHECK (max_concierge_conversations IS NULL OR max_concierge_conversations > 0);

UPDATE subscription_plans p
SET max_concierge_conversations=t.cap
FROM (VALUES ('emprende',500),('impulso',1500),('escala',4000)) AS t(code,cap)
WHERE p.code=t.code;

-- The grandfathered plan keeps Emprende's cap.
UPDATE subscription_plans g
SET max_concierge_conversations=e.max_concierge_conversations
FROM subscription_plans e
WHERE g.code='emprende-abastecimiento' AND e.code='emprende';

-- Global policy: one row. Starts measuring only.
CREATE TABLE concierge_usage_settings (
  singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
  mode text NOT NULL DEFAULT 'measure' CHECK (mode IN ('measure','warn','enforce')),
  warn_percent integer NOT NULL DEFAULT 80 CHECK (warn_percent BETWEEN 1 AND 100),
  updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO concierge_usage_settings(singleton) VALUES (true);

-- Company exception over the plan cap. A row with NULL is unlimited.
CREATE TABLE organization_concierge_limits (
  organization_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
  max_conversations integer CHECK (max_conversations IS NULL OR max_conversations > 0),
  updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Monthly counter: the row lock serializes the cap check per company.
CREATE TABLE concierge_monthly_usage (
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  period_month date NOT NULL CHECK (period_month=date_trunc('month',period_month)::date),
  conversations integer NOT NULL DEFAULT 0 CHECK (conversations >= 0),
  rejected_conversations integer NOT NULL DEFAULT 0 CHECK (rejected_conversations >= 0),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id,period_month)
);

CREATE TABLE concierge_conversations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  location_id uuid NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
  conversation_id text NOT NULL CHECK (length(conversation_id) BETWEEN 1 AND 160),
  kind text NOT NULL CHECK (kind IN ('table','delivery')),
  period_month date NOT NULL,
  started_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id,conversation_id)
);
CREATE INDEX concierge_conversations_period_idx ON concierge_conversations(organization_id,period_month);

-- Tokens per turn and model tier, attributed to the conversation's month.
CREATE TABLE concierge_token_usage (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  conversation_ref uuid NOT NULL REFERENCES concierge_conversations(id) ON DELETE CASCADE,
  period_month date NOT NULL,
  request_id text NOT NULL CHECK (length(request_id) BETWEEN 1 AND 160),
  tier text NOT NULL CHECK (tier IN ('lite','flash')),
  input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
  output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (conversation_ref,request_id,tier)
);
CREATE INDEX concierge_token_usage_period_idx ON concierge_token_usage(organization_id,period_month);
