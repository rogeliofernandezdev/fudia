-- Agregar token QR a las mesas para vincular el proceso de atención
ALTER TABLE tables
  ADD COLUMN IF NOT EXISTS qr_token text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS qr_enabled boolean NOT NULL DEFAULT false;

-- Generar tokens QR únicos para las mesas existentes
UPDATE tables
  SET qr_token = encode(gen_random_bytes(16), 'hex'),
      qr_enabled = true
  WHERE qr_token = '';

-- Índice para búsqueda por token QR (acceso público sin sesión)
CREATE INDEX IF NOT EXISTS tables_qr_token_idx ON tables(qr_token) WHERE qr_token != '';

-- Constraint de unicidad del token QR por organización
ALTER TABLE tables
  ADD CONSTRAINT tables_qr_token_unique UNIQUE (qr_token);
