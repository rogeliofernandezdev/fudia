-- El cupo diario no puede quedar por debajo de las porciones ya comprometidas.
UPDATE product_availability
SET portion_quantity = sold_quantity,
    updated_at = now()
WHERE portion_quantity IS NOT NULL
  AND portion_quantity < sold_quantity;

ALTER TABLE product_availability
  ADD CONSTRAINT product_availability_quota_covers_sold
  CHECK (portion_quantity IS NULL OR portion_quantity >= sold_quantity);
