-- Tolerancia de llegada de las reservas (2026-11-26). Cada local define cuántos
-- minutos se espera al cliente después de la hora reservada; cada reserva copia
-- ese valor al crearse y puede ajustarlo. Vencida la tolerancia sin que la
-- reserva se siente, el barrido del API la pasa a «No asistió» (no_show), marca
-- expired_at y avisa a caja en la campana (kind reservation_no_show).
-- La garantía es opcional y por defecto no hay costo: 0 significa «sin garantía».
-- El local define el importe que toma cada reserva nueva; la reserva puede
-- cambiarlo o quitarlo. Su cobro en Caja llega con la migración 000150.
ALTER TABLE locations
  ADD COLUMN reservation_tolerance_minutes integer NOT NULL DEFAULT 15
  CHECK (reservation_tolerance_minutes BETWEEN 0 AND 240),
  ADD COLUMN reservation_deposit_amount numeric(12,2) NOT NULL DEFAULT 0
  CHECK (reservation_deposit_amount >= 0);

ALTER TABLE reservations
  ADD COLUMN tolerance_minutes integer NOT NULL DEFAULT 15
  CHECK (tolerance_minutes BETWEEN 0 AND 240),
  ADD COLUMN deposit_amount numeric(12,2) NOT NULL DEFAULT 0
  CHECK (deposit_amount >= 0),
  ADD COLUMN expired_at timestamptz,
  ADD CONSTRAINT reservations_expired_no_show_check
  CHECK (expired_at IS NULL OR status='no_show');

-- Las reservas abiertas cuya hora ya pasó con la tolerancia inicial quedan como
-- «No asistió» sin aviso: son historial anterior a esta regla, no ausencias de hoy.
UPDATE reservations
SET status='no_show',expired_at=now(),updated_at=now()
WHERE status IN ('pending','confirmed')
  AND starts_at+make_interval(mins=>tolerance_minutes)<=now();

-- El barrido solo recorre reservas abiertas por hora de inicio.
CREATE INDEX reservations_open_due_idx
  ON reservations(starts_at)
  WHERE status IN ('pending','confirmed');

ALTER TABLE staff_notices DROP CONSTRAINT staff_notices_kind_check;
ALTER TABLE staff_notices ADD CONSTRAINT staff_notices_kind_check
  CHECK (kind IN ('table_order','counter_order','delivery_order','pickup_order','order_ready','bill_requested','assistance_requested','stale_account','delivery_unassigned','delivery_pickup_delayed','pay_detected','reservation_no_show'));
