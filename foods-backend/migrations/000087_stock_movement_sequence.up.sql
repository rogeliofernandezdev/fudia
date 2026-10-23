-- Los movimientos registrados en una misma transacción comparten created_at
-- (now() es el inicio de la transacción) y el Kárdex los ordenaba por un UUID
-- aleatorio. sequence_number fija el orden real de registro: los movimientos
-- existentes se numeran por fecha e id, y los nuevos reciben el siguiente valor.
ALTER TABLE stock_movements ADD COLUMN sequence_number bigint;

UPDATE stock_movements sm
SET sequence_number=ordered.position
FROM (SELECT id,row_number() OVER (ORDER BY created_at,id) AS position FROM stock_movements) ordered
WHERE ordered.id=sm.id;

ALTER TABLE stock_movements
  ALTER COLUMN sequence_number SET NOT NULL,
  ALTER COLUMN sequence_number ADD GENERATED ALWAYS AS IDENTITY;

SELECT setval(
  pg_get_serial_sequence('stock_movements','sequence_number'),
  COALESCE((SELECT max(sequence_number) FROM stock_movements),0)+1,
  false
);
