-- Migra dos productos físicos creados antes del flujo de Inventario.
-- La operación se detiene si los nombres no identifican exactamente un tenant
-- o si alguno ya tiene un estado/historial incompatible.
DO $$
DECLARE
  target_org uuid;
  candidate_count integer;
  changed_count integer;
BEGIN
  SELECT count(*)
  INTO candidate_count
  FROM (
    SELECT organization_id
    FROM products
    WHERE name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
    GROUP BY organization_id
    HAVING count(*) FILTER (WHERE name = 'Cusqueña Dorada 330ml') = 1
       AND count(*) FILTER (WHERE name = 'Inca Kola 500ml') = 1
  ) candidates;

  IF candidate_count <> 1 THEN
    RAISE EXCEPTION
      'legacy physical products migration expected exactly one matching organization, found %',
      candidate_count;
  END IF;

  SELECT organization_id
  INTO target_org
  FROM (
    SELECT organization_id
    FROM products
    WHERE name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
    GROUP BY organization_id
    HAVING count(*) FILTER (WHERE name = 'Cusqueña Dorada 330ml') = 1
       AND count(*) FILTER (WHERE name = 'Inca Kola 500ml') = 1
  ) candidates;

  IF EXISTS (
    SELECT 1
    FROM products
    WHERE organization_id = target_org
      AND name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
      AND NOT active
  ) THEN
    RAISE EXCEPTION 'legacy physical products migration requires both products to be active';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM products
    WHERE organization_id = target_org
      AND name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
      AND quantity_control <> 'none'
  ) THEN
    RAISE EXCEPTION 'legacy physical products migration requires both products to use quantity_control=none';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM menu_combos mc
    JOIN products p
      ON p.id = mc.product_id
     AND p.organization_id = mc.organization_id
    WHERE p.organization_id = target_org
      AND p.name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
  ) THEN
    RAISE EXCEPTION 'legacy physical products migration cannot migrate combo products';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM inventory_items ii
    JOIN products p
      ON p.id = ii.product_id
     AND p.organization_id = ii.organization_id
    WHERE p.organization_id = target_org
      AND p.name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
  ) OR EXISTS (
    SELECT 1
    FROM inventory_entries ie
    JOIN products p
      ON p.id = ie.product_id
     AND p.organization_id = ie.organization_id
    WHERE p.organization_id = target_org
      AND p.name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
  ) OR EXISTS (
    SELECT 1
    FROM stock_movements sm
    JOIN products p
      ON p.id = sm.product_id
     AND p.organization_id = sm.organization_id
    WHERE p.organization_id = target_org
      AND p.name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
  ) THEN
    RAISE EXCEPTION 'legacy physical products migration found existing inventory history';
  END IF;

  UPDATE products
  SET quantity_control = 'inventory',
      updated_at = now()
  WHERE organization_id = target_org
    AND name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
    AND quantity_control = 'none';

  GET DIAGNOSTICS changed_count = ROW_COUNT;

  IF changed_count <> 2 THEN
    RAISE EXCEPTION
      'legacy physical products migration expected to update 2 products, updated %',
      changed_count;
  END IF;
END $$;
