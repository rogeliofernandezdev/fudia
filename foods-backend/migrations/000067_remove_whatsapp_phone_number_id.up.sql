ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_active_configuration_check;

ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_phone_number_id_check;

ALTER TABLE platform_whatsapp_channels
  DROP COLUMN IF EXISTS phone_number_id;
