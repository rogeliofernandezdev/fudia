-- Restaura el formato anterior a partir de los documentos de origen.
UPDATE stock_movements sm
SET note='Recepción '||pr.code||' de '||po.number||CASE WHEN sm.note<>'' THEN ' · '||sm.note ELSE '' END
FROM purchase_receipts pr
JOIN purchase_orders po ON po.id=pr.purchase_order_id AND po.organization_id=pr.organization_id
WHERE sm.source_type='purchase_receipt' AND pr.id=sm.source_id AND pr.organization_id=sm.organization_id
  AND sm.note !~ '^Recepción REC-';

UPDATE stock_movements sm
SET note=r.code||' · '||sm.note
FROM purchase_returns r
WHERE sm.source_type='purchase_return' AND r.id=sm.source_id AND r.organization_id=sm.organization_id
  AND sm.note !~ '^DEV-';

UPDATE stock_movements sm
SET note=CASE WHEN sm.movement_type='transfer_out' THEN t.code||' → '||tl.name ELSE t.code||' desde transferencia' END
FROM inventory_transfers t
JOIN locations tl ON tl.id=t.to_location_id AND tl.organization_id=t.organization_id
WHERE sm.source_type='inventory_transfer' AND t.id=sm.source_id AND t.organization_id=sm.organization_id
  AND sm.note='';
