-- Delivery Fudia se gestiona globalmente: la plataforma define el orden de sus
-- centrales para todos los locales y activa Delivery Fudia por empresa (todos
-- sus locales lo usan). Reemplaza la configuración por local de la 000113.
ALTER TABLE delivery_partners ADD COLUMN priority integer;
UPDATE delivery_partners p SET priority=ranked.n
FROM (SELECT id,row_number() OVER (ORDER BY created_at,id) AS n FROM delivery_partners) ranked
WHERE ranked.id=p.id;
ALTER TABLE delivery_partners ALTER COLUMN priority SET NOT NULL;
ALTER TABLE delivery_partners ADD CONSTRAINT delivery_partners_priority_check CHECK (priority >= 1);

CREATE TABLE organization_delivery_settings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  enabled boolean NOT NULL DEFAULT false,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id)
);

-- Una empresa que ya usaba Delivery Fudia en algún local lo conserva.
INSERT INTO organization_delivery_settings(organization_id,enabled)
SELECT DISTINCT organization_id,true FROM location_delivery_settings WHERE mode='fudia'
ON CONFLICT (organization_id) DO NOTHING;

DROP TABLE location_delivery_partners;
DROP TABLE location_delivery_settings;

-- La prioridad de una oferta es el orden global de la central (sin tope de 3).
ALTER TABLE delivery_dispatch_offers DROP CONSTRAINT delivery_dispatch_offers_priority_check;
ALTER TABLE delivery_dispatch_offers ADD CONSTRAINT delivery_dispatch_offers_priority_check CHECK (priority >= 1);
