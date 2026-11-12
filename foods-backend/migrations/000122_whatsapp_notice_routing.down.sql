DROP TRIGGER IF EXISTS customer_notices_pin_transport ON customer_notices;
DROP TRIGGER IF EXISTS delivery_partner_messages_pin_transport ON delivery_partner_messages;
DROP FUNCTION IF EXISTS pin_customer_notice_transport();
DROP FUNCTION IF EXISTS pin_partner_message_transport();
-- Retener las referencias históricas aunque se reviertan los triggers.
