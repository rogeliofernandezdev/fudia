-- Aviso al cliente de WhatsApp «tu pedido va en camino»: lo genera el paso a
-- en_camino de un pedido de delivery creado por Concierge (por ejemplo, cuando
-- una central de Delivery Fudia pulsa «Recoger»).
ALTER TABLE customer_notices DROP CONSTRAINT customer_notices_kind_check;
ALTER TABLE customer_notices ADD CONSTRAINT customer_notices_kind_check
  CHECK (kind IN ('order_ready','order_sent','order_on_the_way'));
