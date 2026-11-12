ALTER TABLE customer_notices
  ADD COLUMN routing_key text NOT NULL DEFAULT '',
  ADD COLUMN recipient text NOT NULL DEFAULT '';
ALTER TABLE delivery_partner_messages
  ADD COLUMN routing_key text NOT NULL DEFAULT '',
  ADD COLUMN legacy_route boolean NOT NULL DEFAULT true;
ALTER TABLE delivery_partner_messages ALTER COLUMN legacy_route SET DEFAULT false;

CREATE FUNCTION pin_customer_notice_transport() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  SELECT c.routing_key,c.recipient INTO NEW.routing_key,NEW.recipient
  FROM concierge_conversation_origins c
  WHERE c.conversation_id=NEW.conversation_id AND c.organization_id=NEW.organization_id AND c.location_id=NEW.location_id;
  IF NEW.routing_key IS NULL THEN
    -- El productor anterior solo usaba Meta. Se fija su referencia inicial,
    -- nunca el proveedor/número configurado al reclamar el aviso.
    SELECT r.routing_key INTO NEW.routing_key FROM platform_whatsapp_routes r
    JOIN locations l ON l.id=NEW.location_id AND l.organization_id=NEW.organization_id
    JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
    WHERE r.country_code=fp.country_code AND r.legacy ORDER BY r.created_at,r.id LIMIT 1;
  END IF;
  NEW.routing_key=COALESCE(NEW.routing_key,'');
  NEW.recipient=COALESCE(NULLIF(NEW.recipient,''),NEW.customer_phone,'');
  RETURN NEW;
END $$;
CREATE TRIGGER customer_notices_pin_transport BEFORE INSERT ON customer_notices
  FOR EACH ROW EXECUTE FUNCTION pin_customer_notice_transport();

CREATE FUNCTION pin_partner_message_transport() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  SELECT c.routing_key INTO NEW.routing_key FROM delivery_dispatches d
  JOIN concierge_delivery_requests q ON q.order_id=d.order_id AND q.organization_id=d.organization_id AND q.location_id=d.location_id
  JOIN concierge_conversation_origins c ON c.conversation_id=q.conversation_id AND c.organization_id=q.organization_id AND c.location_id=q.location_id
  WHERE d.id=NEW.dispatch_id ORDER BY q.created_at LIMIT 1;
  IF NEW.routing_key IS NULL THEN
    SELECT w.routing_key INTO NEW.routing_key FROM platform_whatsapp_channels w
    JOIN locations l ON l.id=NEW.location_id AND l.organization_id=NEW.organization_id
    JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
    WHERE w.country_code=fp.country_code AND w.active AND w.superseded_by IS NULL;
  END IF;
  NEW.routing_key=COALESCE(NEW.routing_key,'');
  RETURN NEW;
END $$;
CREATE TRIGGER delivery_partner_messages_pin_transport BEFORE INSERT ON delivery_partner_messages
  FOR EACH ROW EXECUTE FUNCTION pin_partner_message_transport();
