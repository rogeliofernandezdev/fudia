ALTER TABLE platform_whatsapp_channels
  ALTER COLUMN phone_number_id DROP NOT NULL;

ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_phone_number_id_check;

ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_phone_number_id_check
  CHECK (phone_number_id IS NULL OR length(btrim(phone_number_id)) > 0);

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

INSERT INTO platform_whatsapp_channels(country_code, phone_number, display_name, active)
VALUES ('PE', '+51914832364', 'FudIA Perú', false)
ON CONFLICT (phone_number) DO NOTHING;
