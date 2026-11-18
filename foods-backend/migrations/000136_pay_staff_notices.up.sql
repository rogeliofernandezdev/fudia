ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account','delivery_unassigned','delivery_pickup_delayed','pay_detected'));

-- Recupera los pagos pendientes recientes cuyo aviso se descartaba al validar
-- el tipo. La clave coincide con PaymentDetected + Normalize; no duplica avisos.
INSERT INTO staff_notices(organization_id,location_id,kind,audience,title,detail,dedupe_key,created_at)
SELECT n.organization_id,n.location_id,'pay_detected','cashier',
       'Pago recibido por ' || CASE n.provider WHEN 'yape' THEN 'Yape' ELSE 'Plin' END,
       left(concat_ws(' · ',n.currency || ' ' || n.amount::text,NULLIF(n.sender_name,'')),240),
       left('pay-detected:' || n.event_key || ':' || n.provider,200),n.created_at
FROM pay_notifications n
WHERE n.status='detected' AND n.created_at>now()-interval '24 hours'
ON CONFLICT (organization_id,dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING;
