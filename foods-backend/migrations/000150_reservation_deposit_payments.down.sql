-- Quitar el cobro de garantías borra las garantías que aún no tienen pedido
-- (y sus devoluciones y movimientos de caja); las aplicadas o retenidas ya son
-- cobros normales de un pedido y se conservan.
DELETE FROM cash_movements WHERE source_type IN ('reservation_deposit','reservation_deposit_refund');
DELETE FROM payment_refunds pr USING payments p
 WHERE pr.payment_id=p.id AND pr.organization_id=p.organization_id AND p.order_id IS NULL;
DELETE FROM payments WHERE order_id IS NULL;
-- Las líneas «Garantía de reserva no utilizada» no tienen producto: sin este
-- tipo no caben en order_items y se eliminan (sus pedidos quedan sin detalle).
DELETE FROM order_items WHERE item_type='reservation_deposit';
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_product_required;
ALTER TABLE order_items ADD CONSTRAINT order_items_product_required CHECK (product_id IS NOT NULL) NOT VALID;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_item_type_check;
ALTER TABLE order_items ADD CONSTRAINT order_items_item_type_check CHECK (item_type IN ('product','combo'));

ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements ADD CONSTRAINT cash_movements_source_type_check
  CHECK(source_type IN ('manual','cash_sale','cash_refund','cash_pull','deposit','transfer_in','transfer_out','adjustment','cash_tip','tip_payout','expense','expense_void'));

DROP INDEX IF EXISTS payments_held_deposits_idx;
DROP INDEX IF EXISTS payments_one_deposit_per_reservation_idx;
ALTER TABLE payments
  DROP CONSTRAINT IF EXISTS payments_deposit_outcome_check,
  DROP CONSTRAINT IF EXISTS payments_order_or_reservation_check,
  DROP CONSTRAINT IF EXISTS payments_reservation_tenant_fk,
  DROP COLUMN IF EXISTS deposit_outcome,
  DROP COLUMN IF EXISTS reservation_id;
ALTER TABLE payments ALTER COLUMN order_id SET NOT NULL;
