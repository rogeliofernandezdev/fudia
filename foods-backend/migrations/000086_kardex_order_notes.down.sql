-- Restaura el formato anterior «Pedido <uuid>» en los movimientos de pedidos,
-- incluidos los registrados con la nota descriptiva después de la migración.
UPDATE stock_movements
SET note='Pedido '||source_id::text
WHERE source_type='order'
  AND (movement_type, note) IN (
    ('sale','Salida automática por venta'),
    ('sale_reversal','Reversa de venta'),
    ('sale_adjustment','Ajuste por edición del pedido')
  );
