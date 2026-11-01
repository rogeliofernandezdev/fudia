-- Facturación electrónica por país (docs/14_EINVOICING.md). El comprobante
-- PENDING se escribe en la transacción del cobro y actúa como outbox: el worker
-- toma las filas PENDING/ERROR vencidas y las envía al proveedor.

CREATE TABLE einvoicing_profile_settings (
  fiscal_profile_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  enabled boolean NOT NULL DEFAULT false,
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_]{1,39}$'),
  environment text NOT NULL DEFAULT 'test' CHECK (environment IN ('test','production')),
  credentials_ref text,
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (fiscal_profile_id,organization_id)
    REFERENCES organization_fiscal_profiles(id,organization_id) ON DELETE CASCADE
);

CREATE TABLE einvoicing_location_settings (
  location_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  default_document text NOT NULL DEFAULT 'none' CHECK (default_document IN ('none','receipt','invoice')),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);

CREATE TABLE einvoicing_series (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  document_type text NOT NULL CHECK (document_type IN ('invoice','receipt','credit_note')),
  series text NOT NULL CHECK (series ~ '^[A-Z0-9]{1,8}$'),
  last_number integer NOT NULL DEFAULT 0 CHECK (last_number >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id,series),
  UNIQUE (location_id,document_type),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id) ON DELETE CASCADE
);

CREATE TABLE fiscal_documents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  fiscal_profile_id uuid NOT NULL,
  order_id uuid NOT NULL,
  country_code char(2) NOT NULL,
  document_type text NOT NULL CHECK (document_type IN ('invoice','receipt','credit_note')),
  series text NOT NULL,
  number integer NOT NULL CHECK (number > 0),
  issue_date date NOT NULL,
  currency char(3) NOT NULL,
  customer_document_type text NOT NULL DEFAULT '',
  customer_document_number text NOT NULL DEFAULT '',
  customer_name text NOT NULL DEFAULT '',
  customer_address text NOT NULL DEFAULT '',
  net_total numeric(14,2) NOT NULL CHECK (net_total >= 0),
  tax_total numeric(14,2) NOT NULL CHECK (tax_total >= 0),
  total numeric(14,2) NOT NULL CHECK (total >= 0),
  status text NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','SENDING','ACCEPTED','OBSERVED','REJECTED','ERROR','CANCELLED')),
  provider text NOT NULL,
  environment text NOT NULL CHECK (environment IN ('test','production')),
  provider_reference text,
  provider_code text,
  provider_message text,
  pdf_url text,
  xml_url text,
  cdr_url text,
  digest_value text,
  attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  accepted_at timestamptz,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id),
  UNIQUE (organization_id,series,number),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id),
  FOREIGN KEY (fiscal_profile_id,organization_id) REFERENCES organization_fiscal_profiles(id,organization_id)
);

-- Un pedido tiene como máximo un comprobante vigente.
CREATE UNIQUE INDEX fiscal_documents_one_per_order_idx ON fiscal_documents(organization_id,order_id)
  WHERE document_type IN ('invoice','receipt') AND status <> 'CANCELLED';
CREATE INDEX fiscal_documents_due_idx ON fiscal_documents(next_attempt_at)
  WHERE status IN ('PENDING','ERROR','SENDING');
CREATE INDEX fiscal_documents_list_idx ON fiscal_documents(organization_id,location_id,created_at DESC);

CREATE TABLE fiscal_document_lines (
  document_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  line_no smallint NOT NULL CHECK (line_no > 0),
  description text NOT NULL,
  unit_code text NOT NULL DEFAULT 'NIU' CHECK (unit_code ~ '^[A-Z0-9]{2,3}$'),
  quantity numeric(10,2) NOT NULL CHECK (quantity > 0),
  unit_price numeric(12,2) NOT NULL CHECK (unit_price >= 0),
  net_amount numeric(14,2) NOT NULL,
  tax_amount numeric(14,2) NOT NULL,
  total numeric(14,2) NOT NULL,
  PRIMARY KEY (document_id,line_no),
  FOREIGN KEY (document_id,organization_id) REFERENCES fiscal_documents(id,organization_id) ON DELETE CASCADE
);

CREATE TABLE fiscal_document_taxes (
  document_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  code text NOT NULL,
  name text NOT NULL,
  rate numeric(7,6) NOT NULL CHECK (rate BETWEEN 0 AND 1),
  amount numeric(14,2) NOT NULL,
  sort_order smallint NOT NULL,
  PRIMARY KEY (document_id,code),
  FOREIGN KEY (document_id,organization_id) REFERENCES fiscal_documents(id,organization_id) ON DELETE CASCADE
);

-- Credenciales cifradas (AES-256-GCM con llave maestra externa) por perfil
-- fiscal y proveedor. El alcance «platform» queda previsto para credenciales
-- compartidas de la plataforma; hoy solo se usa «organization».
CREATE TABLE einvoicing_credentials (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  scope text NOT NULL DEFAULT 'organization' CHECK (scope IN ('organization','platform')),
  organization_id uuid REFERENCES organizations(id) ON DELETE CASCADE,
  fiscal_profile_id uuid,
  provider text NOT NULL CHECK (provider ~ '^[a-z][a-z0-9_]{1,39}$'),
  key_version smallint NOT NULL CHECK (key_version > 0),
  nonce bytea NOT NULL,
  ciphertext bytea NOT NULL,
  metadata jsonb NOT NULL DEFAULT '{}',
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((scope='organization' AND organization_id IS NOT NULL AND fiscal_profile_id IS NOT NULL)
      OR (scope='platform' AND organization_id IS NULL AND fiscal_profile_id IS NULL)),
  FOREIGN KEY (fiscal_profile_id,organization_id)
    REFERENCES organization_fiscal_profiles(id,organization_id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX einvoicing_credentials_profile_idx ON einvoicing_credentials(fiscal_profile_id,provider) WHERE scope='organization';
CREATE UNIQUE INDEX einvoicing_credentials_platform_idx ON einvoicing_credentials(provider) WHERE scope='platform';

-- XML firmado y CDR: archivo legal que el emisor debe conservar.
CREATE TABLE fiscal_document_files (
  document_id uuid NOT NULL,
  organization_id uuid NOT NULL,
  kind text NOT NULL CHECK (kind IN ('xml','cdr')),
  file_name text NOT NULL,
  content_type text NOT NULL,
  content bytea NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (document_id,kind),
  FOREIGN KEY (document_id,organization_id) REFERENCES fiscal_documents(id,organization_id) ON DELETE CASCADE
);
