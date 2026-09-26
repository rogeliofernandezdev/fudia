CREATE TABLE concierge_order_requests (
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  table_id uuid NOT NULL,
  conversation_id text NOT NULL,
  order_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id,location_id,conversation_id),
  UNIQUE (order_id,organization_id),
  FOREIGN KEY (location_id,organization_id)
    REFERENCES locations(id,organization_id),
  FOREIGN KEY (table_id,organization_id,location_id)
    REFERENCES tables(id,organization_id,location_id),
  FOREIGN KEY (order_id,organization_id)
    REFERENCES orders(id,organization_id) ON DELETE CASCADE,
  CHECK (length(btrim(conversation_id)) BETWEEN 1 AND 160)
);

CREATE INDEX concierge_order_requests_table_idx
  ON concierge_order_requests(organization_id,location_id,table_id,created_at DESC);
