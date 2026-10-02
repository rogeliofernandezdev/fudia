-- Revierte la clasificación solo si los productos todavía no recibieron
-- entradas ni movimientos de Inventario después de aplicar la migración.
DO $$
DECLARE
  target_org uuid;
  candidate_count integer;
  legacy_product_count integer;
  changed_count integer;
BEGIN
  SELECT count(*)
  INTO legacy_product_count
  FROM products
  WHERE name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml');

  IF legacy_product_count = 0 THEN
    RAISE NOTICE 'legacy physical products rollback skipped: target products are absent';
    RETURN;
  END IF;

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
      'legacy physical products rollback found target data but expected exactly one matching organization, found %',
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
      AND quantity_control <> 'inventory'
  ) THEN
    RAISE EXCEPTION 'legacy physical products rollback requires both products to use quantity_control=inventory';
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
    RAISE EXCEPTION 'legacy physical products rollback refused because inventory history already exists';
  END IF;

  UPDATE products
  SET quantity_control = 'none',
      updated_at = now()
  WHERE organization_id = target_org
    AND name IN ('Cusqueña Dorada 330ml', 'Inca Kola 500ml')
    AND quantity_control = 'inventory';

  GET DIAGNOSTICS changed_count = ROW_COUNT;

  IF changed_count <> 2 THEN
    RAISE EXCEPTION
      'legacy physical products rollback expected to update 2 products, updated %',
      changed_count;
  END IF;
END $$;
