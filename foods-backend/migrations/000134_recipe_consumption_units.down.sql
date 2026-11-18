-- Refuse a downgrade that would round physical stock or change stored recipes.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM stock_balances WHERE quantity<>round(quantity,3))
 OR EXISTS(SELECT 1 FROM stock_movements WHERE quantity_delta<>round(quantity_delta,3) OR balance_after<>round(balance_after,3))
 OR EXISTS(SELECT 1 FROM inventory_adjustments WHERE stock_before<>round(stock_before,3) OR stock_after<>round(stock_after,3))
 OR EXISTS(SELECT 1 FROM product_recipe_items WHERE quantity<>round(quantity,3) OR consumption_factor<>1) THEN
  RAISE EXCEPTION 'Cannot remove recipe units while converted recipes or high precision stock exist';
 END IF;
END $$;
ALTER TABLE product_recipe_items DROP CONSTRAINT recipe_consumption_equivalence, DROP CONSTRAINT recipe_consumption_unit_fk,
 DROP COLUMN consumption_unit, DROP COLUMN consumption_quantity, DROP COLUMN consumption_factor,
 ALTER COLUMN quantity TYPE numeric(14,3);
ALTER TABLE stock_balances ALTER COLUMN quantity TYPE numeric(14,3);
ALTER TABLE stock_movements ALTER COLUMN quantity_delta TYPE numeric(14,3), ALTER COLUMN balance_after TYPE numeric(14,3);
ALTER TABLE inventory_adjustments ALTER COLUMN stock_before TYPE numeric(14,3), ALTER COLUMN stock_after TYPE numeric(14,3);
DROP TABLE inventory_measurement_units;
