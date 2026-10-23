-- Los movimientos de Kárdex de pedidos guardaban «Pedido <uuid>» como nota y el
-- Kárdex mostraba el identificador interno. El pedido sigue en source_id (y su
-- código como referencia); la nota pasa a describir solo el motivo. Solo se
-- tocan las notas que coinciden exactamente con el formato anterior.
UPDATE stock_movements
SET note=CASE movement_type
  WHEN 'sale_reversal' THEN 'Reversa de venta'
  WHEN 'sale_adjustment' THEN 'Ajuste por edición del pedido'
  ELSE 'Salida automática por venta'
END
WHERE source_type='order'
  AND movement_type IN ('sale','sale_reversal','sale_adjustment')
  AND note='Pedido '||source_id::text;
