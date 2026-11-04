-- Aviso «cuenta anterior sin cerrar»: alguien escaneó el QR de una mesa cuya
-- comanda abierta lleva horas sin rondas nuevas (el cliente anterior pudo irse).
ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account'));
