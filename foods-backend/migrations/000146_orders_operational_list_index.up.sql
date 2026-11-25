-- La bandeja ordena por estado operativo, independientemente del cobro o cierre
-- de la cuenta de Salón. Mantener el índice anterior permite un despliegue gradual.
CREATE INDEX IF NOT EXISTS orders_operational_list_idx
  ON orders(
    organization_id,
    location_id,
    (CASE WHEN status IN ('entregado','cancelado') THEN 1 ELSE 0 END),
    created_at DESC
  );
