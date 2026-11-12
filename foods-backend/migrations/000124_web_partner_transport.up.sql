-- Las centrales siguen en WhatsApp; un destinatario de central nunca es una sesión web.
CREATE OR REPLACE FUNCTION pin_partner_message_transport() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  SELECT c.routing_key INTO NEW.routing_key FROM delivery_dispatches d
  JOIN concierge_delivery_requests q ON q.order_id=d.order_id AND q.organization_id=d.organization_id AND q.location_id=d.location_id
  JOIN concierge_conversation_origins c ON c.conversation_id=q.conversation_id AND c.organization_id=q.organization_id AND c.location_id=q.location_id
  JOIN platform_whatsapp_routes source_route ON source_route.routing_key=c.routing_key AND source_route.provider<>'FUDIA_WEB'
  WHERE d.id=NEW.dispatch_id ORDER BY q.created_at LIMIT 1;
  IF NEW.routing_key IS NULL THEN
    SELECT r.routing_key INTO NEW.routing_key FROM platform_whatsapp_routes r
    JOIN locations l ON l.id=NEW.location_id AND l.organization_id=NEW.organization_id
    JOIN organization_fiscal_profiles fp ON fp.id=l.fiscal_profile_id AND fp.organization_id=l.organization_id
    WHERE r.country_code=fp.country_code AND r.provider IN ('META_CLOUD_API','WHATSMEOW')
    ORDER BY EXISTS(SELECT 1 FROM platform_whatsapp_channels w
      WHERE w.routing_key=r.routing_key AND w.active AND w.superseded_by IS NULL) DESC,
      r.created_at DESC,r.id DESC LIMIT 1;
  END IF;
  NEW.routing_key=COALESCE(NEW.routing_key,'');
  RETURN NEW;
END $$;
