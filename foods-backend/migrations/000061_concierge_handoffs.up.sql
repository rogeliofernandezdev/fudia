CREATE TABLE concierge_handoffs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  table_id uuid NOT NULL,
  conversation_id text NOT NULL CHECK (length(btrim(conversation_id)) BETWEEN 1 AND 160),
  customer_phone text NOT NULL DEFAULT '',
  reason text NOT NULL DEFAULT '' CHECK (length(reason) <= 240),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','resolved')),
  requested_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  resolved_by uuid,
  UNIQUE (id,organization_id,location_id),
  FOREIGN KEY (table_id,organization_id,location_id)
    REFERENCES tables(id,organization_id,location_id),
  FOREIGN KEY (resolved_by,organization_id)
    REFERENCES users(id,organization_id),
  CHECK (
    (status='pending' AND resolved_at IS NULL AND resolved_by IS NULL)
    OR
    (status='resolved' AND resolved_at IS NOT NULL AND resolved_by IS NOT NULL)
  )
);

CREATE UNIQUE INDEX concierge_handoffs_pending_conversation_uq
  ON concierge_handoffs(organization_id,location_id,conversation_id)
  WHERE status='pending';

CREATE INDEX concierge_handoffs_operations_idx
  ON concierge_handoffs(organization_id,location_id,status,requested_at DESC);
