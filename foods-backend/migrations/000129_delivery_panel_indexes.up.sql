-- Índices del panel de la central de Delivery Fudia, que se consulta cada 5 s.
-- Las ofertas aceptadas conservan status='accepted' para siempre, así que el
-- panel parte del despacho de la central: los repartos en curso y los
-- entregados de las últimas horas, sin recorrer el historial.

-- Repartos en curso de una central (panel y paso de motorizado).
CREATE INDEX delivery_dispatches_partner_active_idx
  ON delivery_dispatches(accepted_partner_id)
  WHERE status IN ('accepted','picked_up');

-- Entregados recientes de una central (historial de 12 horas del panel).
CREATE INDEX delivery_dispatches_partner_delivered_idx
  ON delivery_dispatches(accepted_partner_id, delivered_at DESC)
  WHERE status='delivered';

-- Ofertas pendientes de una central, sin las aceptadas históricas.
CREATE INDEX delivery_dispatch_offers_partner_pending_idx
  ON delivery_dispatch_offers(partner_id)
  WHERE status='pending';

-- Limpieza periódica de sesiones vencidas o revocadas de los operadores.
CREATE INDEX delivery_partner_sessions_expires_idx
  ON delivery_partner_sessions(expires_at);
