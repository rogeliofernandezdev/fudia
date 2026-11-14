-- Panel web del proveedor de Delivery Fudia: cada central abre su panel con un
-- enlace privado, confirma la atención, la recogida y la entrega, y comparte la
-- ubicación del repartidor para el seguimiento del cliente en el chat.

-- Enlace privado del panel. PostgreSQL genera el valor; cada fila recibe uno distinto.
ALTER TABLE delivery_partners
  ADD COLUMN panel_token text NOT NULL DEFAULT replace(gen_random_uuid()::text,'-','')
    CHECK (panel_token ~ '^[a-f0-9]{32}$');
CREATE UNIQUE INDEX delivery_partners_panel_token_idx ON delivery_partners(panel_token);

-- Entrega confirmada por la central y última ubicación conocida del repartidor.
ALTER TABLE delivery_dispatches
  ADD COLUMN delivered_at timestamptz,
  ADD COLUMN courier_latitude numeric(9,6),
  ADD COLUMN courier_longitude numeric(9,6),
  ADD COLUMN courier_located_at timestamptz,
  ADD CONSTRAINT delivery_dispatches_courier_location_check CHECK (
    (courier_latitude IS NULL) = (courier_longitude IS NULL)
    AND (courier_latitude IS NULL) = (courier_located_at IS NULL)
    AND (courier_latitude IS NULL OR courier_latitude BETWEEN -90 AND 90)
    AND (courier_longitude IS NULL OR courier_longitude BETWEEN -180 AND 180)
  );

ALTER TABLE delivery_dispatches DROP CONSTRAINT delivery_dispatches_status_check;
ALTER TABLE delivery_dispatches ADD CONSTRAINT delivery_dispatches_status_check
  CHECK (status IN ('searching','accepted','unassigned','picked_up','delivered','own','cancelled'));
ALTER TABLE delivery_dispatches DROP CONSTRAINT delivery_dispatches_check;
ALTER TABLE delivery_dispatches ADD CONSTRAINT delivery_dispatches_check
  CHECK (status NOT IN ('accepted','picked_up','delivered') OR accepted_partner_id IS NOT NULL);

-- El panel lista las ofertas y los repartos en curso de una central.
CREATE INDEX delivery_dispatch_offers_partner_idx
  ON delivery_dispatch_offers(partner_id, status) WHERE status IN ('pending','accepted');
