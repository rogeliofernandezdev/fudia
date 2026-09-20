CREATE TABLE customers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id),
  code text NOT NULL DEFAULT ('CLI-' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 8))),
  customer_type text NOT NULL DEFAULT 'person' CHECK (customer_type IN ('person','company')),
  display_name text NOT NULL,
  document_type text NOT NULL DEFAULT '', document_number text NOT NULL DEFAULT '',
  phone text NOT NULL DEFAULT '', email text NOT NULL DEFAULT '',
  preferred_channel text NOT NULL DEFAULT 'none' CHECK (preferred_channel IN ('none','whatsapp','phone','email')),
  preferences text NOT NULL DEFAULT '', notes text NOT NULL DEFAULT '',
  marketing_consent boolean NOT NULL DEFAULT false, vip_override boolean NOT NULL DEFAULT false,
  visit_count integer NOT NULL DEFAULT 0 CHECK (visit_count >= 0),
  total_spent numeric(14,2) NOT NULL DEFAULT 0 CHECK (total_spent >= 0),
  last_purchase_at timestamptz,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, code), UNIQUE (id, organization_id)
);
CREATE UNIQUE INDEX customers_document_uq ON customers(organization_id, document_type, document_number)
  WHERE document_type <> '' AND document_number <> '';
CREATE INDEX customers_scope_idx ON customers(organization_id, active, display_name);

CREATE TABLE customer_addresses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL, customer_id uuid NOT NULL,
  label text NOT NULL DEFAULT 'Principal', address text NOT NULL, reference text NOT NULL DEFAULT '',
  district text NOT NULL DEFAULT '', city text NOT NULL DEFAULT '', country_code char(2) NOT NULL DEFAULT 'PE',
  is_default boolean NOT NULL DEFAULT false, active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (customer_id, organization_id) REFERENCES customers(id, organization_id) ON DELETE CASCADE
);
CREATE INDEX customer_addresses_scope_idx ON customer_addresses(organization_id, customer_id, active);
