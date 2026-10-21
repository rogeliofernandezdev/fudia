ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_active_configuration_check;

UPDATE platform_whatsapp_channels
SET active=false,updated_at=now()
WHERE active
  AND (secret_ref IS NULL OR length(btrim(secret_ref))=0);

ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_active_configuration_check
  CHECK (
    NOT active OR (
      phone_number_id IS NOT NULL
      AND length(btrim(phone_number_id)) > 0
      AND secret_ref IS NOT NULL
      AND length(btrim(secret_ref)) > 0
    )
  );
