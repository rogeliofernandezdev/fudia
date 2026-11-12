-- No se elimina silenciosamente la historia de un canal utilizado.
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM platform_whatsapp_routes WHERE provider='FUDIA_WEB')
     OR EXISTS (SELECT 1 FROM platform_whatsapp_channels WHERE provider='FUDIA_WEB')
     OR EXISTS (SELECT 1 FROM customer_notices WHERE customer_phone='') THEN
    RAISE EXCEPTION 'Hay canales web o avisos anónimos. No se puede revertir sin archivar sus datos.';
  END IF;
END $$;
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_customer_phone_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_customer_phone_check CHECK (length(btrim(customer_phone)) BETWEEN 1 AND 32);
DROP INDEX platform_whatsapp_channels_current_number_idx;
CREATE UNIQUE INDEX platform_whatsapp_channels_current_number_idx ON platform_whatsapp_channels(phone_number) WHERE superseded_by IS NULL;
ALTER TABLE platform_whatsapp_routes DROP CONSTRAINT platform_whatsapp_routes_provider_check;
ALTER TABLE platform_whatsapp_routes ADD CONSTRAINT platform_whatsapp_routes_provider_check CHECK (provider IN ('META_CLOUD_API','WHATSMEOW'));
ALTER TABLE platform_whatsapp_channels DROP CONSTRAINT platform_whatsapp_channels_phone_number_check;
ALTER TABLE platform_whatsapp_channels ADD CONSTRAINT platform_whatsapp_channels_phone_number_check CHECK (length(btrim(phone_number))>0);
ALTER TABLE platform_whatsapp_channels DROP CONSTRAINT platform_whatsapp_channels_provider_check;
ALTER TABLE platform_whatsapp_channels ADD CONSTRAINT platform_whatsapp_channels_provider_check CHECK (provider IN ('META_CLOUD_API','WHATSMEOW'));
