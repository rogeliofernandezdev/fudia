-- Cobro de la garantía de reserva en Caja (2026-11-26).
--
-- La garantía es un cobro real (tabla payments) del turno de caja que la
-- recibe: el efectivo entra al cuadre y Yape, Plin o tarjeta suman a su medio
-- en ese turno. Mientras el cliente no llega, el cobro no tiene pedido
-- (order_id NULL) y queda ligado a la reserva (reservation_id). Después:
--   applied  → el cliente llegó: el cobro pasa al pedido de su mesa y descuenta
--              el saldo; la boleta del consumo sale por el total.
--   retained → no llegó (o se canceló reteniéndola): el cobro pasa a un pedido
--              de mostrador «Garantía de reserva no utilizada» con su boleta.
-- Devolverla registra una devolución normal (payment_refunds) sobre el cobro.
-- Una reserva tiene como máximo una garantía cobrada.
ALTER TABLE payments ALTER COLUMN order_id DROP NOT NULL;

ALTER TABLE payments
  ADD COLUMN reservation_id uuid,
  ADD COLUMN deposit_outcome text,
  ADD CONSTRAINT payments_reservation_tenant_fk
    FOREIGN KEY (reservation_id, organization_id) REFERENCES reservations(id, organization_id),
  ADD CONSTRAINT payments_order_or_reservation_check
    CHECK (order_id IS NOT NULL OR reservation_id IS NOT NULL),
  ADD CONSTRAINT payments_deposit_outcome_check
    CHECK (deposit_outcome IS NULL OR (reservation_id IS NOT NULL AND order_id IS NOT NULL AND deposit_outcome IN ('applied','retained')));

CREATE UNIQUE INDEX payments_one_deposit_per_reservation_idx
  ON payments(organization_id, reservation_id) WHERE reservation_id IS NOT NULL;

-- Garantías cobradas que esperan al cliente: Cobros las ofrece y el barrido
-- retiene las de reservas que pasaron a «No asistió».
CREATE INDEX payments_held_deposits_idx
  ON payments(organization_id, location_id) WHERE reservation_id IS NOT NULL AND order_id IS NULL;

-- La venta de una garantía retenida tiene una sola línea sin producto, de tipo
-- reservation_deposit; toda otra línea sigue exigiendo su producto.
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_item_type_check;
ALTER TABLE order_items ADD CONSTRAINT order_items_item_type_check
  CHECK (item_type IN ('product','combo','reservation_deposit'));
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_product_required;
ALTER TABLE order_items ADD CONSTRAINT order_items_product_required
  CHECK (product_id IS NOT NULL OR item_type='reservation_deposit') NOT VALID;

ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements ADD CONSTRAINT cash_movements_source_type_check
  CHECK(source_type IN ('manual','cash_sale','cash_refund','cash_pull','deposit','transfer_in','transfer_out','adjustment','cash_tip','tip_payout','expense','expense_void','reservation_deposit','reservation_deposit_refund'));
