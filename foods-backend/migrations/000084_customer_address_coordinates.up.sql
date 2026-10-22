-- Ubicación opcional de cada dirección de cliente, marcada en el mapa del admin.
ALTER TABLE customer_addresses
  ADD COLUMN latitude numeric(9,6),
  ADD COLUMN longitude numeric(9,6),
  ADD CONSTRAINT customer_addresses_point_check CHECK (
    (latitude IS NULL) = (longitude IS NULL)
    AND (latitude IS NULL OR latitude BETWEEN -90 AND 90)
    AND (longitude IS NULL OR longitude BETWEEN -180 AND 180)
  );
