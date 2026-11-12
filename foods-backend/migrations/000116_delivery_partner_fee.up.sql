-- Delivery Fudia: la central paga al restaurante solo el costo del pedido y
-- cobra al cliente el pedido más su delivery. Al aceptar, el costo de envío se
-- guarda en el despacho (partner_fee) y sale del pedido del restaurante
-- (orders.delivery_fee=0, total=subtotal), así caja, comprobante y reportes
-- cuadran con lo que realmente cobra el local. fee_released indica que Orders
-- ya lo descontó; si el local vuelve a su delivery propio, se devuelve.
ALTER TABLE delivery_dispatches
  ADD COLUMN partner_fee numeric(12,2) CHECK (partner_fee IS NULL OR partner_fee >= 0),
  ADD COLUMN fee_released boolean NOT NULL DEFAULT false;
