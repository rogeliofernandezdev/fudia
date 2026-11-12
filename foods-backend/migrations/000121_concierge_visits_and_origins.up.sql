CREATE TABLE concierge_customer_identities (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  identity_key text NOT NULL UNIQUE CHECK (identity_key ~ '^[0-9a-f]{64}$'),
  verified_phone text NOT NULL DEFAULT '',
  canonical_id uuid REFERENCES concierge_customer_identities(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (canonical_id IS NULL OR canonical_id<>id)
);
CREATE INDEX concierge_customer_identity_canonical_idx ON concierge_customer_identities(canonical_id);

CREATE TABLE concierge_venue_visits (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  identity_id uuid NOT NULL REFERENCES concierge_customer_identities(id),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  location_id uuid NOT NULL,
  customer_id uuid REFERENCES customers(id),
  last_visited_at timestamptz NOT NULL DEFAULT now(),
  visit_count bigint NOT NULL DEFAULT 1 CHECK (visit_count>0),
  UNIQUE(identity_id,location_id),
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id)
);

CREATE TABLE concierge_conversation_origins (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id text NOT NULL UNIQUE,
  organization_id uuid NOT NULL REFERENCES organizations(id),
  location_id uuid NOT NULL,
  routing_key text NOT NULL REFERENCES platform_whatsapp_routes(routing_key),
  recipient text NOT NULL CHECK (length(recipient) BETWEEN 1 AND 160),
  identity_id uuid NOT NULL REFERENCES concierge_customer_identities(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id)
);
CREATE INDEX concierge_conversation_origins_venue_idx ON concierge_conversation_origins(organization_id,location_id);
