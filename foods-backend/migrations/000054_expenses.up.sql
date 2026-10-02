CREATE TABLE expense_categories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id)
);

CREATE UNIQUE INDEX expense_categories_name_uq
  ON expense_categories(organization_id,lower(name));
CREATE INDEX expense_categories_scope_idx
  ON expense_categories(organization_id,active,name);

CREATE TABLE expenses (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  location_id uuid NOT NULL,
  category_id uuid NOT NULL,
  business_date date NOT NULL,
  description text NOT NULL,
  amount numeric(14,2) NOT NULL CHECK (amount > 0),
  payment_method text NOT NULL CHECK (payment_method IN ('cash','bank_transfer','card','digital_wallet','other')),
  reference text NOT NULL DEFAULT '',
  notes text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','void')),
  created_by uuid NOT NULL,
  voided_by uuid,
  voided_at timestamptz,
  void_reason text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id,organization_id),
  FOREIGN KEY (location_id,organization_id) REFERENCES locations(id,organization_id),
  FOREIGN KEY (category_id,organization_id) REFERENCES expense_categories(id,organization_id),
  FOREIGN KEY (created_by,organization_id) REFERENCES users(id,organization_id),
  FOREIGN KEY (voided_by,organization_id) REFERENCES users(id,organization_id),
  CHECK (
    (status='active' AND voided_by IS NULL AND voided_at IS NULL AND void_reason='')
    OR
    (status='void' AND voided_by IS NOT NULL AND voided_at IS NOT NULL AND length(btrim(void_reason))>=4)
  )
);

CREATE INDEX expenses_scope_date_idx
  ON expenses(organization_id,location_id,business_date DESC,created_at DESC);
CREATE INDEX expenses_category_idx
  ON expenses(organization_id,location_id,category_id,business_date DESC);
CREATE INDEX expenses_status_idx
  ON expenses(organization_id,location_id,status,business_date DESC);

UPDATE subscription_plans
SET module_keys=array_append(module_keys,'costos'),updated_at=now()
WHERE code IN ('legacy','escala') AND NOT ('costos'=ANY(module_keys));

UPDATE organization_modules om
SET active=true,updated_at=now()
FROM organization_subscriptions os
JOIN subscription_plans sp ON sp.id=os.plan_id
WHERE om.organization_id=os.organization_id
  AND om.module_key='costos'
  AND os.status IN ('trial','active','past_due')
  AND 'costos'=ANY(sp.module_keys);

UPDATE roles
SET menu_access=CASE WHEN 'costos'=ANY(menu_access) THEN menu_access ELSE array_append(menu_access,'costos') END,
    permissions=array_cat(
      permissions,
      ARRAY(
        SELECT value
        FROM unnest(ARRAY['expenses.read','expenses.manage','expenses.void']::text[]) value
        WHERE NOT (value=ANY(permissions))
      )
    ),
    updated_at=now()
WHERE system_key IN ('administrator','location_manager','accounting');

UPDATE roles
SET menu_access=CASE WHEN 'costos'=ANY(menu_access) THEN menu_access ELSE array_append(menu_access,'costos') END,
    permissions=CASE WHEN 'expenses.read'=ANY(permissions) THEN permissions ELSE array_append(permissions,'expenses.read') END,
    updated_at=now()
WHERE system_key='auditor';
