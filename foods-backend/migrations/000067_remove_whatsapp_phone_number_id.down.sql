ALTER TABLE platform_whatsapp_channels
  ADD COLUMN IF NOT EXISTS phone_number_id text;

UPDATE platform_whatsapp_channels
SET active=false,updated_at=now()
WHERE active;

ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_phone_number_id_key UNIQUE (phone_number_id);

ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_phone_number_id_check
  CHECK (phone_number_id IS NULL OR length(btrim(phone_number_id)) > 0);

ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_active_configuration_check
  CHECK (
    NOT active OR (
      phone_number_id IS NOT NULL
      AND length(btrim(phone_number_id)) > 0
    )
  );
