-- Restaura el conjunto único de plantillas y la activación original.
UPDATE payment_methods SET active=false,updated_at=now() WHERE code IN ('yape','plin');
UPDATE payment_methods SET active=true,updated_at=now() WHERE code IN ('cash','card','transfer','wallet','other');

DELETE FROM payment_method_templates WHERE country_code<>'';
UPDATE payment_method_templates SET active=true WHERE code='other';
ALTER TABLE payment_method_templates DROP CONSTRAINT payment_method_templates_country_check;
ALTER TABLE payment_method_templates DROP CONSTRAINT payment_method_templates_pkey;
ALTER TABLE payment_method_templates DROP COLUMN country_code;
ALTER TABLE payment_method_templates ADD PRIMARY KEY (code);
