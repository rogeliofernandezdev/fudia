ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_active_configuration_check;

ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_active_configuration_check
  CHECK (
    NOT active OR (
      phone_number_id IS NOT NULL
      AND length(btrim(phone_number_id)) > 0
    )
  );
