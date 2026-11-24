-- Las asignaciones retiradas por la subida no se restauran: sin fila por local,
-- los listados anteriores vuelven a mostrar el artículo con el mínimo de empresa.
DROP TRIGGER IF EXISTS stock_movement_assigns_inventory_item ON stock_movements;
DROP FUNCTION IF EXISTS assign_inventory_item_on_movement();
DROP INDEX IF EXISTS inventory_location_settings_scope_idx;
ALTER TABLE inventory_location_settings DROP COLUMN IF EXISTS active;
