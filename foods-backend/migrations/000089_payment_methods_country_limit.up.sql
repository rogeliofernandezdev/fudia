-- Medios de pago por país y como máximo 4 activos por empresa.
-- Perú opera con Efectivo, Tarjeta, Yape y Plin; country_code='' es el
-- conjunto general para el resto de países.
ALTER TABLE payment_method_templates ADD COLUMN country_code text NOT NULL DEFAULT '';
ALTER TABLE payment_method_templates DROP CONSTRAINT payment_method_templates_pkey;
ALTER TABLE payment_method_templates ADD PRIMARY KEY (country_code,code);
ALTER TABLE payment_method_templates
  ADD CONSTRAINT payment_method_templates_country_check CHECK (country_code='' OR country_code ~ '^[A-Z]{2}$');

-- El conjunto general queda en 4 activos: Otro se conserva inactivo.
UPDATE payment_method_templates SET active=false WHERE country_code='' AND code='other';

INSERT INTO payment_method_templates(
  country_code,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
) VALUES
  ('PE','cash','Efectivo','Movimiento físico de efectivo',true,true,true,true,10),
  ('PE','card','Tarjeta','Pago mediante tarjeta o POS',true,true,true,false,20),
  ('PE','yape','Yape','Pago con Yape',true,true,true,false,30),
  ('PE','plin','Plin','Pago con Plin',true,true,true,false,40);

-- País de cada empresa: su perfil fiscal predeterminado.
CREATE TEMP TABLE payment_method_org_country ON COMMIT DROP AS
SELECT o.id AS organization_id,
       COALESCE((SELECT fp.country_code::text FROM organization_fiscal_profiles fp
                 WHERE fp.organization_id=o.id ORDER BY fp.is_default DESC,fp.created_at LIMIT 1),
                o.country_code::text) AS country_code
FROM organizations o;

-- Empresas peruanas: incorporan Yape y Plin y dejan inactivos Transferencia,
-- Billetera digital y Otro (el historial de cobros y gastos se conserva).
INSERT INTO payment_methods(
  organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
)
SELECT c.organization_id,t.code,t.name,t.description,t.active,t.sales_enabled,t.expenses_enabled,t.affects_cash,t.sort_order
FROM payment_method_org_country c
JOIN payment_method_templates t ON t.country_code='PE'
WHERE c.country_code='PE'
ON CONFLICT (organization_id,code) DO NOTHING;

UPDATE payment_methods pm
SET active=false,updated_at=now()
FROM payment_method_org_country c
WHERE pm.organization_id=c.organization_id AND c.country_code='PE'
  AND pm.code IN ('transfer','wallet','other') AND pm.active;

-- Ninguna empresa queda con más de 4 activos: se conservan los de menor orden.
UPDATE payment_methods pm
SET active=false,updated_at=now()
FROM (
  SELECT organization_id,code,row_number() OVER (PARTITION BY organization_id ORDER BY sort_order,code) AS position
  FROM payment_methods WHERE active
) ranked
WHERE pm.organization_id=ranked.organization_id AND pm.code=ranked.code AND ranked.position>4;
