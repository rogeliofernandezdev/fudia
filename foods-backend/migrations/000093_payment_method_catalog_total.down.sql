-- Restore the general templates from migration 89. Company catalogs are unchanged.
UPDATE payment_method_templates
SET active=true
WHERE country_code='' AND code IN ('transfer','wallet');
