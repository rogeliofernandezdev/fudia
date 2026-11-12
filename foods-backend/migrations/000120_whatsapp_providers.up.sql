-- Un único registro operativo por país. Los duplicados históricos se conservan
-- íntegros como registros archivados; nunca se elige entre varios canales activos.
ALTER TABLE platform_whatsapp_channels
  ADD COLUMN provider text NOT NULL DEFAULT 'META_CLOUD_API'
    CHECK (provider IN ('META_CLOUD_API','WHATSMEOW')),
  ADD COLUMN provider_key text NOT NULL DEFAULT '',
  ADD COLUMN routing_key text NOT NULL DEFAULT '',
  ADD COLUMN revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  ADD COLUMN verified_at timestamptz,
  ADD COLUMN superseded_by uuid REFERENCES platform_whatsapp_channels(id),
  ADD COLUMN needs_reconciliation boolean NOT NULL DEFAULT false;

-- La migración 000067 retiró los identificadores Meta de esta tabla.
-- El activo original se importa mediante el servicio autenticado desde su
-- configuración autorizada, verificando su número antes de usarlo.


CREATE TABLE platform_whatsapp_channel_history (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  channel_id uuid NOT NULL REFERENCES platform_whatsapp_channels(id),
  actor_id uuid REFERENCES users(id),
  action text NOT NULL,
  before_config jsonb,
  after_config jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX platform_whatsapp_channel_history_channel_idx
  ON platform_whatsapp_channel_history(channel_id,created_at DESC,id);

-- Referencias inmutables de transporte, no números operativos adicionales.
-- Los avisos antiguos nunca consultan el número vigente para elegir remitente.
CREATE TABLE platform_whatsapp_routes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  routing_key text NOT NULL UNIQUE,
  country_code char(2) NOT NULL REFERENCES platform_countries(code),
  provider text NOT NULL CHECK (provider IN ('META_CLOUD_API','WHATSMEOW')),
  provider_key text NOT NULL CHECK (length(provider_key)>0),
  phone_number text NOT NULL,
  legacy boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO platform_whatsapp_routes(routing_key,country_code,provider,provider_key,phone_number,legacy)
SELECT routing_key,country_code,provider,provider_key,phone_number,true
FROM platform_whatsapp_channels WHERE routing_key<>'';

INSERT INTO platform_whatsapp_channel_history(channel_id,action,after_config)
SELECT id,'migration.imported',jsonb_build_object(
  'countryCode',country_code,'phoneNumber',phone_number,'displayName',display_name,
  'provider',provider,'active',active,'revision',revision)
FROM platform_whatsapp_channels;

WITH ranked AS (
  SELECT id,country_code,
    first_value(id) OVER (PARTITION BY country_code ORDER BY active DESC,created_at,id) AS canonical_id,
    count(*) OVER (PARTITION BY country_code) AS records,
    count(*) FILTER (WHERE active) OVER (PARTITION BY country_code) AS enabled
  FROM platform_whatsapp_channels
)
UPDATE platform_whatsapp_channels w SET
  superseded_by=CASE WHEN w.id<>r.canonical_id THEN r.canonical_id END,
  active=CASE WHEN r.enabled>1 OR w.id<>r.canonical_id THEN false ELSE w.active END,
  needs_reconciliation=w.needs_reconciliation OR r.enabled>1,
  updated_at=now()
FROM ranked r WHERE r.id=w.id AND r.records>1;

UPDATE platform_whatsapp_channels SET needs_reconciliation=needs_reconciliation OR active,active=false;

CREATE UNIQUE INDEX platform_whatsapp_channels_one_country_idx
  ON platform_whatsapp_channels(country_code) WHERE superseded_by IS NULL;
ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_phone_number_key;
CREATE UNIQUE INDEX platform_whatsapp_channels_current_number_idx
  ON platform_whatsapp_channels(phone_number) WHERE superseded_by IS NULL;
CREATE UNIQUE INDEX platform_whatsapp_channels_current_provider_idx
  ON platform_whatsapp_channels(provider,provider_key)
  WHERE superseded_by IS NULL AND provider_key<>'';
ALTER TABLE platform_whatsapp_channels
  DROP CONSTRAINT IF EXISTS platform_whatsapp_channels_active_configuration_check;
ALTER TABLE platform_whatsapp_channels
  ADD CONSTRAINT platform_whatsapp_channels_active_configuration_check
    CHECK (NOT active OR (provider_key<>'' AND routing_key<>'' AND NOT needs_reconciliation));

INSERT INTO platform_whatsapp_channel_history(channel_id,action,after_config)
SELECT id,'migration.reconciled',jsonb_build_object(
  'countryCode',country_code,'phoneNumber',phone_number,'active',active,
  'archived',superseded_by IS NOT NULL,'needsReconciliation',needs_reconciliation)
FROM platform_whatsapp_channels
WHERE superseded_by IS NOT NULL OR needs_reconciliation;
