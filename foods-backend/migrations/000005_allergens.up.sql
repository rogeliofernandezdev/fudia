CREATE TABLE IF NOT EXISTS allergens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL UNIQUE,
  icon text NOT NULL DEFAULT ''
);

INSERT INTO allergens (name) VALUES
  ('Gluten'), ('Lactosa'), ('Mariscos'), ('Frutos secos'),
  ('Huevo'), ('Soya'), ('Pescado'), ('Sésamo'),
  ('Mostaza'), ('Apio'), ('Sulfitos'), ('Cacahuete')
ON CONFLICT (name) DO NOTHING;
