CREATE TABLE payment_methods (
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  code text NOT NULL,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  active boolean NOT NULL DEFAULT true,
  sales_enabled boolean NOT NULL DEFAULT true,
  expenses_enabled boolean NOT NULL DEFAULT true,
  affects_cash boolean NOT NULL DEFAULT false,
  sort_order integer NOT NULL DEFAULT 100,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id,code),
  CHECK (code ~ '^[a-z0-9][a-z0-9_-]{0,39}$')
);

CREATE INDEX payment_methods_usage_idx
  ON payment_methods(organization_id,active,sales_enabled,expenses_enabled,sort_order);

INSERT INTO payment_methods(
  organization_id,code,name,description,active,sales_enabled,expenses_enabled,affects_cash,sort_order
)
SELECT o.id,v.code,v.name,v.description,true,v.sales_enabled,v.expenses_enabled,v.affects_cash,v.sort_order
FROM organizations o
CROSS JOIN (VALUES
  ('cash','Efectivo','Movimiento físico de efectivo',true,true,true,10),
  ('card','Tarjeta','Pago mediante tarjeta o POS',true,true,false,20),
  ('transfer','Transferencia bancaria','Transferencia a cuenta bancaria',true,true,false,30),
  ('wallet','Billetera digital','Yape, Plin u otra billetera digital',true,true,false,40),
  ('other','Otro','Otro medio autorizado por la empresa',true,true,false,90)
) AS v(code,name,description,sales_enabled,expenses_enabled,affects_cash,sort_order);

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_check;
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_payment_method_check;

UPDATE expenses SET payment_method='transfer' WHERE payment_method='bank_transfer';
UPDATE expenses SET payment_method='wallet' WHERE payment_method='digital_wallet';

ALTER TABLE payments
  ADD CONSTRAINT payments_payment_method_fk
  FOREIGN KEY (organization_id,method)
  REFERENCES payment_methods(organization_id,code);

ALTER TABLE expenses
  ADD CONSTRAINT expenses_payment_method_fk
  FOREIGN KEY (organization_id,payment_method)
  REFERENCES payment_methods(organization_id,code);
