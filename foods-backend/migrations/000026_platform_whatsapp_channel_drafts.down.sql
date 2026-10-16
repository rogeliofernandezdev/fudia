DELETE FROM platform_whatsapp_channels
WHERE country_code='PE' AND phone_number='+51914832364' AND phone_number_id IS NULL;

ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_active_configuration_check;
ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_phone_number_id_check;
ALTER TABLE platform_whatsapp_channels
  ALTER COLUMN phone_number_id SET NOT NULL;
ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_phone_number_id_check CHECK (length(btrim(phone_number_id)) > 0);
