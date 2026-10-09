ALTER TABLE platform_whatsapp_channels
  ADD COLUMN IF NOT EXISTS secret_ref text;
