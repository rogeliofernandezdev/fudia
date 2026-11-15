-- Vuelve al enlace privado de la central: cada central recibe un enlace nuevo.
ALTER TABLE delivery_partners
  ADD COLUMN panel_token text NOT NULL DEFAULT replace(gen_random_uuid()::text,'-','')
    CHECK (panel_token ~ '^[a-f0-9]{32}$');
CREATE UNIQUE INDEX delivery_partners_panel_token_idx ON delivery_partners(panel_token);

DROP TABLE IF EXISTS delivery_partner_operator_actions;
DROP TABLE IF EXISTS delivery_partner_sessions;
DROP TABLE IF EXISTS delivery_partner_operators;
