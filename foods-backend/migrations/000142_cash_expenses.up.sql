-- Un gasto en efectivo y el egreso del turno son un solo registro (2026-11-22).
-- Con el módulo `costos`, el egreso del turno crea el gasto y su movimiento de
-- caja (source_type 'expense', source_id = gasto). Anular ese gasto con el
-- turno abierto devuelve el importe con un ingreso 'expense_void'. Los egresos
-- manuales anteriores quedan como historial, sin gasto asociado.
ALTER TABLE expenses ADD COLUMN cash_shift_id uuid;
ALTER TABLE expenses
  ADD CONSTRAINT expenses_cash_shift_fk
  FOREIGN KEY (cash_shift_id,organization_id,location_id)
  REFERENCES cash_shifts(id,organization_id,location_id);
CREATE INDEX expenses_cash_shift_idx
  ON expenses(organization_id,location_id,cash_shift_id)
  WHERE cash_shift_id IS NOT NULL;

ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements ADD CONSTRAINT cash_movements_source_type_check
 CHECK(source_type IN ('manual','cash_sale','cash_refund','cash_pull','deposit','transfer_in','transfer_out','adjustment','cash_tip','tip_payout','expense','expense_void'));
