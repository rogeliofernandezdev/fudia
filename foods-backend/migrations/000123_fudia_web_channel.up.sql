-- El chat propio no requiere un teléfono. Las rutas históricas se conservan.
ALTER TABLE platform_whatsapp_channels DROP CONSTRAINT platform_whatsapp_channels_provider_check;
ALTER TABLE platform_whatsapp_channels ADD CONSTRAINT platform_whatsapp_channels_provider_check
  CHECK (provider IN ('META_CLOUD_API','WHATSMEOW','FUDIA_WEB'));
ALTER TABLE platform_whatsapp_channels DROP CONSTRAINT platform_whatsapp_channels_phone_number_check;
ALTER TABLE platform_whatsapp_channels ADD CONSTRAINT platform_whatsapp_channels_phone_number_check
  CHECK ((provider='FUDIA_WEB' AND phone_number='') OR (provider<>'FUDIA_WEB' AND length(btrim(phone_number))>0));
ALTER TABLE platform_whatsapp_routes DROP CONSTRAINT platform_whatsapp_routes_provider_check;
ALTER TABLE platform_whatsapp_routes ADD CONSTRAINT platform_whatsapp_routes_provider_check
  CHECK (provider IN ('META_CLOUD_API','WHATSMEOW','FUDIA_WEB'));
DROP INDEX platform_whatsapp_channels_current_number_idx;
CREATE UNIQUE INDEX platform_whatsapp_channels_current_number_idx
  ON platform_whatsapp_channels(phone_number) WHERE superseded_by IS NULL AND phone_number<>'';
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_customer_phone_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_customer_phone_check
  CHECK (length(btrim(customer_phone)) BETWEEN 1 AND 32 OR (customer_phone='' AND routing_key<>'' AND recipient<>''));
