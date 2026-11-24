-- Artículos por local (2026-11-24). La ficha (nombre y unidad) sigue siendo de la
-- empresa; una fila en inventory_location_settings significa que el local usa el
-- artículo y active=false lo oculta solo en ese local, conservando su kárdex.
ALTER TABLE inventory_location_settings
  ADD COLUMN active boolean NOT NULL DEFAULT true;

CREATE INDEX inventory_location_settings_scope_idx
  ON inventory_location_settings(organization_id,location_id,active);

-- Uso real por local: cualquier documento que movió o pidió el artículo allí.
CREATE TEMP TABLE inventory_location_usage ON COMMIT DROP AS
SELECT DISTINCT organization_id,location_id,inventory_item_id FROM (
  SELECT organization_id,location_id,inventory_item_id FROM stock_balances
  UNION ALL
  SELECT organization_id,location_id,inventory_item_id FROM stock_movements WHERE inventory_item_id IS NOT NULL
  UNION ALL
  SELECT organization_id,location_id,inventory_item_id FROM inventory_entries WHERE inventory_item_id IS NOT NULL
  UNION ALL
  SELECT organization_id,location_id,inventory_item_id FROM inventory_adjustments
  UNION ALL
  SELECT po.organization_id,po.location_id,poi.inventory_item_id
  FROM purchase_order_items poi
  JOIN purchase_orders po ON po.id=poi.purchase_order_id AND po.organization_id=poi.organization_id
  UNION ALL
  SELECT t.organization_id,t.from_location_id,ti.inventory_item_id
  FROM inventory_transfer_items ti
  JOIN inventory_transfers t ON t.id=ti.transfer_id AND t.organization_id=ti.organization_id
  UNION ALL
  SELECT t.organization_id,t.to_location_id,ti.inventory_item_id
  FROM inventory_transfer_items ti
  JOIN inventory_transfers t ON t.id=ti.transfer_id AND t.organization_id=ti.organization_id
) usage;

-- Todo local que ya usó un artículo queda asignado a él.
INSERT INTO inventory_location_settings(
  organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock
)
SELECT u.organization_id,u.location_id,ii.id,ii.minimum_stock,ii.minimum_stock,ii.minimum_stock
FROM inventory_location_usage u
JOIN inventory_items ii ON ii.id=u.inventory_item_id AND ii.organization_id=u.organization_id
ON CONFLICT(location_id,inventory_item_id) DO NOTHING;

-- La migración 42 asignó cada artículo a todos los locales. Si el artículo ya se
-- usa en algún local, se retira de los locales que nunca lo usaron. Un artículo
-- sin ningún uso conserva sus asignaciones porque no se sabe dónde nació.
DELETE FROM inventory_location_settings s
WHERE NOT EXISTS (
    SELECT 1 FROM inventory_location_usage u
    WHERE u.location_id=s.location_id AND u.inventory_item_id=s.inventory_item_id
  )
  AND EXISTS (
    SELECT 1 FROM inventory_location_usage u
    WHERE u.organization_id=s.organization_id AND u.inventory_item_id=s.inventory_item_id
  );

-- Cualquier movimiento de stock en un local asigna (o reactiva) el artículo allí:
-- recepciones, transferencias recibidas, consumo de recetas y reversas.
CREATE FUNCTION assign_inventory_item_on_movement() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.inventory_item_id IS NOT NULL THEN
    INSERT INTO inventory_location_settings(
      organization_id,location_id,inventory_item_id,minimum_stock,reorder_point,optimal_stock,active
    )
    SELECT NEW.organization_id,NEW.location_id,ii.id,ii.minimum_stock,ii.minimum_stock,ii.minimum_stock,true
    FROM inventory_items ii
    WHERE ii.id=NEW.inventory_item_id AND ii.organization_id=NEW.organization_id
    ON CONFLICT(location_id,inventory_item_id) DO UPDATE
      SET active=true,updated_at=now()
      WHERE NOT inventory_location_settings.active;
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER stock_movement_assigns_inventory_item
AFTER INSERT ON stock_movements
FOR EACH ROW EXECUTE FUNCTION assign_inventory_item_on_movement();
