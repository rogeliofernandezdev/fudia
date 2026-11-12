CREATE TABLE IF NOT EXISTS gateway_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  country_code char(2) NOT NULL CHECK (country_code ~ '^[A-Z]{2}$'),
  device_jid text UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS gateway_inbound_outbox (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  sequence bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  session_id uuid NOT NULL REFERENCES gateway_sessions(id),
  event_key text NOT NULL,
  payload jsonb NOT NULL,
  attempts integer NOT NULL DEFAULT 0,
  delivered_at timestamptz,
  UNIQUE (session_id,event_key)
);
CREATE INDEX IF NOT EXISTS gateway_inbound_pending_idx ON gateway_inbound_outbox(sequence) WHERE delivered_at IS NULL;
CREATE TABLE IF NOT EXISTS gateway_outgoing (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id uuid NOT NULL REFERENCES gateway_sessions(id),
  callback_ref text NOT NULL,
  external_id text NOT NULL,
  payload_hash text NOT NULL,
  buttons jsonb NOT NULL DEFAULT '[]',
  state text NOT NULL CHECK (state IN ('sending','sent','uncertain','delivered','read','failed')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (session_id,callback_ref),
  UNIQUE (session_id,external_id)
);
