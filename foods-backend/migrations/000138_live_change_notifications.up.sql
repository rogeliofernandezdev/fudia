-- Avisos de cambios para las pantallas en vivo (SSE en GET /v1/admin/events).
-- Cada escritura en una tabla operativa emite pg_notify('foods_live',
-- 'empresa:local:tema'). PostgreSQL entrega los avisos al confirmar la
-- transacción y descarta los repetidos con el mismo texto dentro de ella, así que
-- un pedido con diez líneas avisa una sola vez por tema. El aviso no lleva datos:
-- la pantalla vuelve a leer por la API, con sus permisos de siempre.
--
-- TG_ARGV[0] es el tema. TG_ARGV[1] indica de dónde sale el local:
--   'row'   → columna location_id de la fila;
--   'order' → el local del pedido de la fila (order_id);
--   'org'   → sin local: avisa a todos los locales de la empresa.
CREATE OR REPLACE FUNCTION notify_live_change() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
  organization uuid;
  location uuid;
BEGIN
  IF TG_OP = 'DELETE' THEN
    organization := OLD.organization_id;
    IF TG_ARGV[1] = 'row' THEN
      location := OLD.location_id;
    ELSIF TG_ARGV[1] = 'order' THEN
      SELECT o.location_id INTO location FROM orders o WHERE o.id = OLD.order_id;
    END IF;
  ELSE
    organization := NEW.organization_id;
    IF TG_ARGV[1] = 'row' THEN
      location := NEW.location_id;
    ELSIF TG_ARGV[1] = 'order' THEN
      SELECT o.location_id INTO location FROM orders o WHERE o.id = NEW.order_id;
    END IF;
  END IF;
  PERFORM pg_notify('foods_live', organization::text || ':' || COALESCE(location::text, '') || ':' || TG_ARGV[0]);
  RETURN NULL;
END;
$$;

CREATE TRIGGER orders_live AFTER INSERT OR UPDATE OR DELETE ON orders
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('orders', 'row');
CREATE TRIGGER order_service_items_live AFTER INSERT OR UPDATE OR DELETE ON order_service_items
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('kitchen', 'order');
CREATE TRIGGER order_kitchen_rounds_live AFTER INSERT OR UPDATE OR DELETE ON order_kitchen_rounds
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('kitchen', 'row');
CREATE TRIGGER payments_live AFTER INSERT OR UPDATE OR DELETE ON payments
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('payments', 'row');
CREATE TRIGGER payment_refunds_live AFTER INSERT OR UPDATE OR DELETE ON payment_refunds
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('payments', 'row');
CREATE TRIGGER cash_shifts_live AFTER INSERT OR UPDATE OR DELETE ON cash_shifts
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('cash', 'row');
CREATE TRIGGER cash_movements_live AFTER INSERT OR UPDATE OR DELETE ON cash_movements
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('cash', 'row');
CREATE TRIGGER staff_notices_live AFTER INSERT OR UPDATE OR DELETE ON staff_notices
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('notices', 'row');
CREATE TRIGGER pay_notifications_live AFTER INSERT OR UPDATE OR DELETE ON pay_notifications
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('pay', 'row');
CREATE TRIGGER product_availability_live AFTER INSERT OR UPDATE OR DELETE ON product_availability
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('availability', 'row');
CREATE TRIGGER stock_balances_live AFTER INSERT OR UPDATE OR DELETE ON stock_balances
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('availability', 'row');
CREATE TRIGGER products_live AFTER INSERT OR UPDATE OR DELETE ON products
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('catalog', 'org');
CREATE TRIGGER delivery_dispatches_live AFTER INSERT OR UPDATE OR DELETE ON delivery_dispatches
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('delivery', 'row');
CREATE TRIGGER tables_live AFTER INSERT OR UPDATE OR DELETE ON tables
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('salon', 'row');
CREATE TRIGGER reservations_live AFTER INSERT OR UPDATE OR DELETE ON reservations
  FOR EACH ROW EXECUTE FUNCTION notify_live_change('reservations', 'row');
