-- Restaura exactamente las fechas anteriores guardadas por la migración.
DO $$
DECLARE
  target_product uuid;
  target_org uuid;
  migration_audit_id uuid;
  backup jsonb;
  item jsonb;
BEGIN
  SELECT a.id, a.organization_id, a.entity_id, a.metadata
  INTO migration_audit_id, target_org, target_product, backup
  FROM audit_log a
  JOIN products p
    ON p.id = a.entity_id
   AND p.organization_id = a.organization_id
  WHERE a.action = 'migration.000028.inventory_date_assigned'
    AND a.entity_type = 'product'
    AND p.name = 'Agua Mineral Angel 700ml'
  ORDER BY a.created_at DESC
  LIMIT 1;

  IF migration_audit_id IS NULL THEN
    IF NOT EXISTS (
      SELECT 1
      FROM products
      WHERE name = 'Agua Mineral Angel 700ml'
    ) THEN
      RAISE NOTICE 'inventory date rollback skipped: target product is absent';
      RETURN;
    END IF;
    RAISE EXCEPTION 'inventory date rollback backup not found';
  END IF;

  UPDATE products
  SET updated_at = (backup->>'previousProductUpdatedAt')::timestamptz
  WHERE id = target_product
    AND organization_id = target_org;

  FOR item IN
    SELECT value
    FROM jsonb_array_elements(backup->'previousStockBalances')
  LOOP
    UPDATE stock_balances
    SET updated_at = (item->>'updatedAt')::timestamptz
    WHERE organization_id = target_org
      AND location_id = (item->>'locationId')::uuid
      AND inventory_item_id = (item->>'inventoryItemId')::uuid;
  END LOOP;

  DELETE FROM audit_log
  WHERE id = migration_audit_id;
END $$;
