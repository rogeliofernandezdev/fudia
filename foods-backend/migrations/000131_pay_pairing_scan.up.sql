-- Vinculación de Fudia Pay como WhatsApp Web: la app avisa en cuanto escanea
-- el QR y Caja cambia el QR por «Vinculando…» antes de que termine el canje.
ALTER TABLE pay_pairing_codes
  ADD COLUMN scanned_at timestamptz,
  ADD COLUMN scanned_model text CHECK (scanned_model IS NULL OR length(scanned_model) <= 120);
