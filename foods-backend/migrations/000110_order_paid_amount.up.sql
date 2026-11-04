-- Cobrado neto del pedido (cobros menos devoluciones), mantenido por PostgreSQL.
-- El listado de POS (cada 30 s en POS y Caja) calculaba este valor con dos
-- subconsultas por cada pedido no cancelado del local para filtrar «por cobrar» y
-- obtener el total: 5,3 s con un año de historia. Los cobros y devoluciones siguen
-- siendo los documentos de origen; esta columna es un derivado que los triggers
-- recalculan por completo (no por incrementos) en la misma transacción que los
-- modifica, así que nunca diverge. Los caminos de cobro ya bloquean el pedido con
-- FOR UPDATE antes de insertar, por lo que el trigger no agrega bloqueos nuevos.
-- `numeric` sin escala conserva el texto anterior: «0» sin cobros y «60.00» con ellos.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS paid_amount numeric NOT NULL DEFAULT 0;

CREATE OR REPLACE FUNCTION refresh_order_paid_amount(target_order uuid, target_organization uuid)
RETURNS void
LANGUAGE sql
AS $$
  UPDATE orders o
  SET paid_amount = COALESCE((
    SELECT sum(p.amount - COALESCE((
      SELECT sum(pr.amount) FROM payment_refunds pr
      WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND pr.location_id=p.location_id
    ),0))
    FROM payments p
    WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.location_id=o.location_id
  ),0)
  WHERE o.id=target_order AND o.organization_id=target_organization;
$$;

CREATE OR REPLACE FUNCTION payments_refresh_order_paid_amount()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP IN ('UPDATE','DELETE') THEN
    PERFORM refresh_order_paid_amount(OLD.order_id, OLD.organization_id);
  END IF;
  IF TG_OP IN ('INSERT','UPDATE') THEN
    PERFORM refresh_order_paid_amount(NEW.order_id, NEW.organization_id);
  END IF;
  RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION payment_refunds_refresh_order_paid_amount()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  refund_order uuid;
BEGIN
  IF TG_OP IN ('UPDATE','DELETE') THEN
    SELECT p.order_id INTO refund_order FROM payments p
    WHERE p.id=OLD.payment_id AND p.organization_id=OLD.organization_id;
    IF refund_order IS NOT NULL THEN
      PERFORM refresh_order_paid_amount(refund_order, OLD.organization_id);
    END IF;
  END IF;
  IF TG_OP IN ('INSERT','UPDATE') THEN
    SELECT p.order_id INTO refund_order FROM payments p
    WHERE p.id=NEW.payment_id AND p.organization_id=NEW.organization_id;
    IF refund_order IS NOT NULL THEN
      PERFORM refresh_order_paid_amount(refund_order, NEW.organization_id);
    END IF;
  END IF;
  RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS payments_paid_amount ON payments;
CREATE TRIGGER payments_paid_amount
  AFTER INSERT OR DELETE OR UPDATE OF amount, order_id, organization_id, location_id ON payments
  FOR EACH ROW EXECUTE FUNCTION payments_refresh_order_paid_amount();

DROP TRIGGER IF EXISTS payment_refunds_paid_amount ON payment_refunds;
CREATE TRIGGER payment_refunds_paid_amount
  AFTER INSERT OR DELETE OR UPDATE OF amount, payment_id, organization_id, location_id ON payment_refunds
  FOR EACH ROW EXECUTE FUNCTION payment_refunds_refresh_order_paid_amount();

-- Carga inicial: solo los pedidos con cobros (el resto ya vale 0 por defecto).
UPDATE orders o
SET paid_amount = totals.net_paid
FROM (
  SELECT p.order_id, p.organization_id, p.location_id,
         sum(p.amount - COALESCE(r.refunded,0)) AS net_paid
  FROM payments p
  LEFT JOIN (
    SELECT payment_id, organization_id, location_id, sum(amount) AS refunded
    FROM payment_refunds
    GROUP BY payment_id, organization_id, location_id
  ) r ON r.payment_id=p.id AND r.organization_id=p.organization_id AND r.location_id=p.location_id
  GROUP BY p.order_id, p.organization_id, p.location_id
) totals
WHERE o.id=totals.order_id AND o.organization_id=totals.organization_id AND o.location_id=totals.location_id;

-- «Por cobrar» del POS (filtro por defecto): pedidos no cancelados con saldo,
-- recorridos del más reciente al más antiguo.
CREATE INDEX IF NOT EXISTS orders_pos_unpaid_idx
  ON orders(organization_id, location_id, created_at DESC, id DESC)
  WHERE status<>'cancelado' AND paid_amount < total;
