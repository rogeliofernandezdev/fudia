ALTER TABLE platform_countries
DROP CONSTRAINT IF EXISTS platform_countries_calling_code_check;

ALTER TABLE platform_countries
DROP COLUMN IF EXISTS calling_code;
