CREATE TABLE concierge_settings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  location_id uuid NOT NULL,
  whatsapp_phone text NOT NULL DEFAULT '',
  active boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id,location_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE,
  CONSTRAINT concierge_settings_whatsapp_phone_check CHECK (
    whatsapp_phone='' OR whatsapp_phone ~ '^\\+[1-9][0-9]{7,14}$'
  )
);

CREATE INDEX concierge_settings_active_idx
  ON concierge_settings(organization_id,location_id)
  WHERE active;
