-- Asigna una fecha de actualización válida al producto legado
-- "Agua Mineral Angel 700ml" y conserva los timestamps previos en auditoría
-- para permitir una reversión exacta.
DO $$
DECLARE
  target_product uuid;
  target_org uuid;
  previous_product_updated_at timestamptz;
  previous_balances jsonb;
  matches integer;
BEGIN
  SELECT count(*)
  INTO matches
  FROM products
  WHERE name = 'Agua Mineral Angel 700ml';

  IF matches <> 1 THEN
    RAISE EXCEPTION
      'inventory date migration expected exactly one Agua Mineral Angel 700ml product, found %',
      matches;
  END IF;

  SELECT id, organization_id, updated_at
  INTO target_product, target_org, previous_product_updated_at
  FROM products
  WHERE name = 'Agua Mineral Angel 700ml';

  SELECT COALESCE(
    jsonb_agg(
      jsonb_build_object(
        'locationId', sb.location_id,
        'inventoryItemId', sb.inventory_item_id,
        'updatedAt', sb.updated_at
      )
      ORDER BY sb.location_id, sb.inventory_item_id
    ),
    '[]'::jsonb
  )
  INTO previous_balances
  FROM stock_balances sb
  JOIN inventory_items ii
    ON ii.id = sb.inventory_item_id
   AND ii.organization_id = sb.organization_id
  WHERE ii.organization_id = target_org
    AND ii.product_id = target_product;

  INSERT INTO audit_log(
    organization_id,
    action,
    entity_type,
    entity_id,
    metadata
  )
  VALUES(
    target_org,
    'migration.000028.inventory_date_assigned',
    'product',
    target_product,
    jsonb_build_object(
      'previousProductUpdatedAt', previous_product_updated_at,
      'previousStockBalances', previous_balances
    )
  );

  UPDATE products
  SET updated_at = now()
  WHERE id = target_product
    AND organization_id = target_org;

  UPDATE stock_balances sb
  SET updated_at = now()
  FROM inventory_items ii
  WHERE ii.id = sb.inventory_item_id
    AND ii.organization_id = sb.organization_id
    AND ii.organization_id = target_org
    AND ii.product_id = target_product;
END $$;
