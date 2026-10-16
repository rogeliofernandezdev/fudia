-- Código legible para documentos de entrada de inventario.
-- Kárdex usa esta referencia en lugar de exponer UUID internos.
ALTER TABLE inventory_entries
  ADD COLUMN code text NOT NULL DEFAULT ('ENT-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 8)));

CREATE UNIQUE INDEX inventory_entries_code_uq
  ON inventory_entries(organization_id, code);
