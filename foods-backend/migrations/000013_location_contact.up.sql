ALTER TABLE locations
  ADD COLUMN phone text NOT NULL DEFAULT '',
  ADD COLUMN opening_hours text NOT NULL DEFAULT '';
