DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM inventory_entries WHERE product_id IS NULL)
     OR EXISTS (SELECT 1 FROM stock_movements WHERE product_id IS NULL) THEN
    RAISE EXCEPTION 'cannot rollback ingredient inventory while product-less entries or movements exist';
  END IF;
END $$;

ALTER TABLE stock_movements
  ALTER COLUMN product_id SET NOT NULL;

ALTER TABLE inventory_entries
  ALTER COLUMN product_id SET NOT NULL;

ALTER TABLE inventory_items
  DROP COLUMN updated_at,
  DROP COLUMN created_at;
