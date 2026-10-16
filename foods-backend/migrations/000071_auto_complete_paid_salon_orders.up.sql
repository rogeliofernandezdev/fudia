-- Reconcile accounts delivered and fully paid before automatic closure existed.
-- Pending delivery, partial payments and refunded balances remain open.
UPDATE orders o
SET completed_at=now(),updated_at=now()
WHERE o.channel='salon' AND o.status='entregado' AND o.completed_at IS NULL
  AND COALESCE((
    SELECT sum(p.amount-COALESCE((
      SELECT sum(pr.amount) FROM payment_refunds pr
      WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id
        AND pr.location_id=p.location_id
    ),0))
    FROM payments p
    WHERE p.order_id=o.id AND p.organization_id=o.organization_id
      AND p.location_id=o.location_id
  ),0) >= o.total;
