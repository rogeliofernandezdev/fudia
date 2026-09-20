CREATE TABLE product_availability (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  location_id uuid NOT NULL REFERENCES locations(id),
  product_id uuid NOT NULL REFERENCES products(id),
  business_date date NOT NULL,
  daily_quota integer CHECK (daily_quota IS NULL OR daily_quota >= 0),
  sold_quantity integer NOT NULL DEFAULT 0 CHECK (sold_quantity >= 0),
  manual_status text NOT NULL DEFAULT 'available'
    CHECK (manual_status IN ('available','low','sold_out')),
  note text NOT NULL DEFAULT '',
  updated_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, location_id, product_id, business_date)
);

CREATE INDEX product_availability_location_day_idx
  ON product_availability (organization_id, location_id, business_date);

COMMENT ON TABLE product_availability IS
  'Estado operativo diario del producto por local; el override manual prevalece sobre el cálculo.';
