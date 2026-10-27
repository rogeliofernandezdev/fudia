-- New non-Peruvian catalogs start with Efectivo and Tarjeta only.
-- Existing company catalogs and all payment/expense history remain untouched.
UPDATE payment_method_templates
SET active=false
WHERE country_code='' AND code IN ('transfer','wallet','other');
