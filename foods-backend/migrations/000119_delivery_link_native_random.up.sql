-- Corrige también bases que ya aplicaron la 115 original. Los QR existentes
-- conservan su token; solo cambia la generación de los siguientes enlaces.
ALTER TABLE location_delivery_links
  ALTER COLUMN qr_token SET DEFAULT replace(gen_random_uuid()::text,'-','');
