DROP TABLE IF EXISTS customer_marketing_consents;
ALTER TABLE tables DROP CONSTRAINT IF EXISTS tables_service_type_check;
ALTER TABLE tables DROP COLUMN IF EXISTS service_type;
