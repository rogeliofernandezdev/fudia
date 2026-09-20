ALTER TABLE products
  ADD COLUMN available_from timestamptz,
  ADD COLUMN available_until timestamptz,
  ADD COLUMN available_days integer[],
  ADD COLUMN available_until_time time;

COMMENT ON COLUMN products.available_from IS 'Fecha/hora desde la que el producto está disponible. NULL = siempre.';
COMMENT ON COLUMN products.available_until IS 'Fecha/hora hasta la que el producto está disponible. NULL = sin fin.';
COMMENT ON COLUMN products.available_days IS 'Días de la semana disponibles (0=Dom,1=Lun,...6=Sab). NULL = todos los días.';
COMMENT ON COLUMN products.available_until_time IS 'Hora límite diario de disponibilidad. NULL = todo el día.';
