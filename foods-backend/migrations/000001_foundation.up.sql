CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE organizations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  legal_name text NOT NULL, trade_name text NOT NULL, tax_id varchar(11) NOT NULL UNIQUE,
  currency char(3) NOT NULL DEFAULT 'PEN', timezone text NOT NULL DEFAULT 'America/Lima',
  active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE locations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id),
  name text NOT NULL, code text NOT NULL, address text NOT NULL DEFAULT '', active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(organization_id,code), UNIQUE(id,organization_id)
);
CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id),
  email text NOT NULL, full_name text NOT NULL, password_hash text NOT NULL, active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(organization_id,email), UNIQUE(id,organization_id)
);
CREATE TABLE roles (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id), name text NOT NULL, permissions text[] NOT NULL DEFAULT '{}', UNIQUE(organization_id,name));
CREATE TABLE user_roles (user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE, location_id uuid REFERENCES locations(id), PRIMARY KEY(user_id,role_id,location_id));
CREATE TABLE sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), token_hash bytea NOT NULL UNIQUE, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  organization_id uuid NOT NULL REFERENCES organizations(id), location_id uuid NOT NULL REFERENCES locations(id),
  expires_at timestamptz NOT NULL, revoked_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_lookup_idx ON sessions(token_hash) WHERE revoked_at IS NULL;

CREATE TABLE menu_categories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id), name text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0, active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(organization_id,name), UNIQUE(id,organization_id)
);
CREATE TABLE products (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id), category_id uuid,
  sku text NOT NULL, name text NOT NULL, description text NOT NULL DEFAULT '', price numeric(12,2) NOT NULL CHECK(price>=0),
  active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(organization_id,sku), UNIQUE(id,organization_id), FOREIGN KEY(category_id,organization_id) REFERENCES menu_categories(id,organization_id)
);
CREATE TABLE inventory_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id), sku text NOT NULL,
  name text NOT NULL, unit text NOT NULL, minimum_stock numeric(14,3) NOT NULL DEFAULT 0, active boolean NOT NULL DEFAULT true,
  UNIQUE(organization_id,sku), UNIQUE(id,organization_id)
);
CREATE TABLE stock_balances (
  organization_id uuid NOT NULL, location_id uuid NOT NULL, inventory_item_id uuid NOT NULL, quantity numeric(14,3) NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(location_id,inventory_item_id),
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id), FOREIGN KEY(inventory_item_id,organization_id) REFERENCES inventory_items(id,organization_id)
);
CREATE TABLE suppliers (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id), tax_id varchar(11), name text NOT NULL, email text, phone text, active boolean NOT NULL DEFAULT true, UNIQUE(id,organization_id));
CREATE TABLE purchase_orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL REFERENCES organizations(id), location_id uuid NOT NULL,
  supplier_id uuid NOT NULL, number text NOT NULL, status text NOT NULL CHECK(status IN ('draft','pending_approval','approved','received','cancelled')),
  total numeric(12,2) NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id), FOREIGN KEY(supplier_id,organization_id) REFERENCES suppliers(id,organization_id), UNIQUE(organization_id,number)
);
CREATE TABLE audit_log (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(), organization_id uuid NOT NULL, location_id uuid, user_id uuid,
  action text NOT NULL, entity_type text NOT NULL, entity_id uuid, reason text, metadata jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX products_scope_idx ON products(organization_id,active,name);
CREATE INDEX inventory_scope_idx ON inventory_items(organization_id,active,name);
CREATE INDEX purchases_scope_idx ON purchase_orders(organization_id,location_id,created_at DESC);
