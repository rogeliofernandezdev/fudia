CREATE TABLE product_modifier_groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  product_id uuid NOT NULL,
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
  required boolean NOT NULL DEFAULT false,
  min_selections integer NOT NULL DEFAULT 0 CHECK (min_selections >= 0),
  max_selections integer NOT NULL DEFAULT 1 CHECK (max_selections >= 1 AND max_selections >= min_selections),
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  FOREIGN KEY(product_id,organization_id)
    REFERENCES products(id,organization_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX product_modifier_groups_name_uq
  ON product_modifier_groups(organization_id,product_id,lower(name));

CREATE INDEX product_modifier_groups_product_idx
  ON product_modifier_groups(organization_id,product_id,sort_order,id);

CREATE TABLE product_modifier_options (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  group_id uuid NOT NULL,
  name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 80),
  surcharge numeric(12,2) NOT NULL DEFAULT 0 CHECK (surcharge >= 0),
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(id,organization_id),
  FOREIGN KEY(group_id,organization_id)
    REFERENCES product_modifier_groups(id,organization_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX product_modifier_options_name_uq
  ON product_modifier_options(organization_id,group_id,lower(name));

CREATE INDEX product_modifier_options_group_idx
  ON product_modifier_options(organization_id,group_id,sort_order,id);

CREATE TABLE order_item_modifier_selections (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  order_item_id uuid NOT NULL,
  group_id uuid NOT NULL,
  group_name text NOT NULL,
  option_id uuid NOT NULL,
  option_name text NOT NULL,
  surcharge numeric(12,2) NOT NULL DEFAULT 0 CHECK (surcharge >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY(order_item_id,organization_id)
    REFERENCES order_items(id,organization_id) ON DELETE CASCADE,
  UNIQUE(order_item_id,group_id,option_id)
);

CREATE INDEX order_item_modifier_selections_item_idx
  ON order_item_modifier_selections(organization_id,order_item_id);
