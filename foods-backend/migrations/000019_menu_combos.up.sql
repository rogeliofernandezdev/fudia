CREATE TABLE menu_combos (
  product_id uuid PRIMARY KEY REFERENCES products(id),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE menu_combo_groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  combo_product_id uuid NOT NULL REFERENCES menu_combos(product_id) ON DELETE CASCADE,
  name text NOT NULL,
  required boolean NOT NULL DEFAULT true,
  min_selections integer NOT NULL DEFAULT 1 CHECK (min_selections >= 0),
  max_selections integer NOT NULL DEFAULT 1 CHECK (max_selections >= min_selections),
  sort_order integer NOT NULL DEFAULT 0
);
CREATE TABLE menu_combo_options (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  group_id uuid NOT NULL REFERENCES menu_combo_groups(id) ON DELETE CASCADE,
  option_product_id uuid NOT NULL REFERENCES products(id),
  surcharge numeric(12,2) NOT NULL DEFAULT 0 CHECK (surcharge >= 0),
  sort_order integer NOT NULL DEFAULT 0,
  UNIQUE(group_id, option_product_id)
);
CREATE INDEX menu_combo_groups_product_idx ON menu_combo_groups(organization_id,combo_product_id);
CREATE INDEX menu_combo_options_group_idx ON menu_combo_options(organization_id,group_id);
