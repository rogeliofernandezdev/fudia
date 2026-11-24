-- Devuelve los nombres anteriores. Los accesos agregados al contador (Ventas,
-- Comprobantes y orders.read) se conservan: no se distingue si ya los tenía.
UPDATE roles r
SET name='Contabilidad',description='Consulta cierres, ventas, gastos e información fiscal',updated_at=now()
WHERE r.system_key='accounting' AND r.name='Contador'
  AND NOT EXISTS (SELECT 1 FROM roles o WHERE o.organization_id=r.organization_id AND o.name='Contabilidad');

UPDATE roles r
SET name='Compras',updated_at=now()
WHERE r.system_key='buyer' AND r.name='Comprador'
  AND NOT EXISTS (SELECT 1 FROM roles o WHERE o.organization_id=r.organization_id AND o.name='Compras');
