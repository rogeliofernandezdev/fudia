-- Tips are staff funds, separate from payment revenue and operating expenses.
CREATE TABLE tip_settings (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES organizations(id),
 location_id uuid REFERENCES locations(id),
 mode text NOT NULL CHECK(mode IN ('disabled','suggest','collect')),
 percentage numeric(6,2) NOT NULL CHECK(percentage>=0 AND percentage<=100),
 show_suggestion boolean NOT NULL DEFAULT false,
 allow_fixed boolean NOT NULL DEFAULT true,
 channels text[] NOT NULL DEFAULT ARRAY['salon'],
 updated_by uuid NOT NULL REFERENCES users(id),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(cardinality(channels)>0 AND channels <@ ARRAY['salon','recojo','delivery','whatsapp']::text[]),
 FOREIGN KEY(location_id,organization_id) REFERENCES locations(id,organization_id)
);
CREATE UNIQUE INDEX tip_settings_scope_uq ON tip_settings(organization_id,COALESCE(location_id,'00000000-0000-0000-0000-000000000000'::uuid));

CREATE TABLE payment_tips (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES organizations(id),
 location_id uuid NOT NULL,
 payment_id uuid NOT NULL UNIQUE,
 order_id uuid NOT NULL,
 shift_id uuid NOT NULL,
 kind text NOT NULL CHECK(kind IN ('percentage','fixed')),
 value numeric(14,2) NOT NULL CHECK(value>0),
 base_amount numeric(14,2) NOT NULL CHECK(base_amount>0),
 amount numeric(14,2) NOT NULL CHECK(amount>0),
 method text NOT NULL,
 affects_cash boolean NOT NULL,
 direct_to_staff boolean NOT NULL DEFAULT false,
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(organization_id,location_id,order_id),
 UNIQUE(id,organization_id,location_id),
 FOREIGN KEY(payment_id,organization_id,location_id) REFERENCES payments(id,organization_id,location_id),
 FOREIGN KEY(shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
 FOREIGN KEY(order_id,organization_id) REFERENCES orders(id,organization_id),
 FOREIGN KEY(organization_id,method) REFERENCES payment_methods(organization_id,code)
);
CREATE INDEX payment_tips_shift_idx ON payment_tips(organization_id,location_id,shift_id);

CREATE TABLE tip_payouts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 organization_id uuid NOT NULL REFERENCES organizations(id),
 location_id uuid NOT NULL,
 shift_id uuid NOT NULL,
 recipient_id uuid NOT NULL REFERENCES users(id),
 amount numeric(14,2) NOT NULL CHECK(amount>0),
 method text NOT NULL CHECK(method IN ('cash','transfer')),
 note text NOT NULL DEFAULT '',
 request_key text NOT NULL CHECK(length(request_key) BETWEEN 8 AND 100),
 created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(organization_id,location_id,request_key),
 UNIQUE(id,organization_id,location_id),
 FOREIGN KEY(shift_id,organization_id,location_id) REFERENCES cash_shifts(id,organization_id,location_id),
 FOREIGN KEY(recipient_id,organization_id) REFERENCES users(id,organization_id)
);
CREATE INDEX tip_payouts_shift_idx ON tip_payouts(organization_id,location_id,shift_id);
CREATE TABLE tip_payout_allocations (
 organization_id uuid NOT NULL,
 location_id uuid NOT NULL,
 payout_id uuid NOT NULL,
 tip_id uuid NOT NULL,
 amount numeric(14,2) NOT NULL CHECK(amount>0),
 PRIMARY KEY(payout_id,tip_id),
 FOREIGN KEY(payout_id,organization_id,location_id) REFERENCES tip_payouts(id,organization_id,location_id),
 FOREIGN KEY(tip_id,organization_id,location_id) REFERENCES payment_tips(id,organization_id,location_id)
);
CREATE INDEX tip_payout_allocations_tip_idx ON tip_payout_allocations(tip_id);
ALTER TABLE cash_movements DROP CONSTRAINT IF EXISTS cash_movements_source_type_check;
ALTER TABLE cash_movements ADD CONSTRAINT cash_movements_source_type_check
 CHECK(source_type IN ('manual','cash_sale','cash_refund','cash_pull','deposit','transfer_in','transfer_out','adjustment','cash_tip','tip_payout'));
