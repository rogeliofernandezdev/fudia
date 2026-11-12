ALTER TABLE delivery_dispatch_offers DROP CONSTRAINT delivery_dispatch_offers_priority_check;
DELETE FROM delivery_dispatch_offers WHERE priority > 3;
ALTER TABLE delivery_dispatch_offers ADD CONSTRAINT delivery_dispatch_offers_priority_check CHECK (priority BETWEEN 1 AND 3);

CREATE TABLE location_delivery_settings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  mode text NOT NULL DEFAULT 'own' CHECK (mode IN ('own','fudia')),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (location_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);
CREATE TABLE location_delivery_partners (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  partner_id uuid NOT NULL REFERENCES delivery_partners(id),
  priority smallint NOT NULL CHECK (priority BETWEEN 1 AND 3),
  UNIQUE (location_id,priority),
  UNIQUE (location_id,partner_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);
INSERT INTO location_delivery_settings(organization_id,location_id,mode)
SELECT l.organization_id,l.id,'fudia' FROM locations l
JOIN organization_delivery_settings s ON s.organization_id=l.organization_id AND s.enabled;

DROP TABLE organization_delivery_settings;
ALTER TABLE delivery_partners DROP CONSTRAINT delivery_partners_priority_check;
ALTER TABLE delivery_partners DROP COLUMN priority;
