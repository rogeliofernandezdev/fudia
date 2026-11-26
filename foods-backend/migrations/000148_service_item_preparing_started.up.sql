-- Hora en que Cocina o Barra empezó a preparar cada plato. Con el tiempo de
-- preparación del producto da el tiempo restante del plato; no se reinicia al
-- devolver un plato de Listo a preparación.
ALTER TABLE order_service_items ADD COLUMN preparing_started_at timestamptz;
UPDATE order_service_items SET preparing_started_at=updated_at WHERE status='preparando';
