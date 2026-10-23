-- Las notas del Kárdex repetían códigos internos generados por la base
-- (REC-…, OC-…, DEV-…, TRF-…). La referencia legible del movimiento ya
-- identifica la recepción, el proveedor o la transferencia; la nota conserva
-- solo lo que escribió el usuario. Solo se tocan notas con el formato anterior.
UPDATE stock_movements
SET note=regexp_replace(note,'^Recepción REC-[0-9A-F]{8} de OC-[0-9A-F]{8}( · )?','')
WHERE source_type='purchase_receipt'
  AND note ~ '^Recepción REC-[0-9A-F]{8} de OC-[0-9A-F]{8}';

UPDATE stock_movements
SET note=regexp_replace(note,'^DEV-[0-9A-F]{8} · ','')
WHERE source_type='purchase_return'
  AND note ~ '^DEV-[0-9A-F]{8} · ';

UPDATE stock_movements
SET note=''
WHERE source_type='inventory_transfer'
  AND (note ~ '^TRF-[0-9A-F]{8} → ' OR note ~ '^TRF-[0-9A-F]{8} desde transferencia$');
