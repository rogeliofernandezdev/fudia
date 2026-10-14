-- Upload reservations have no UPLOADING state; only attached images are visible.
CREATE TABLE product_images (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id),
    location_id uuid NOT NULL REFERENCES locations(id),
    uploaded_by uuid NOT NULL REFERENCES users(id),
    product_id uuid REFERENCES products(id),
    original_key text NOT NULL DEFAULT '',
    content_type text NOT NULL CHECK (content_type IN ('image/jpeg','image/png','image/webp')),
    size_bytes bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 5242880),
    width integer NOT NULL CHECK (width BETWEEN 1 AND 4000),
    height integer NOT NULL CHECK (height BETWEEN 1 AND 4000),
    status text NOT NULL DEFAULT 'PROCESSING' CHECK (status IN ('PROCESSING','READY','FAILED')),
    generation bigint,
    confirmed_at timestamptz,
    expires_at timestamptz NOT NULL,
    lease_until timestamptz,
    attempts integer NOT NULL DEFAULT 0,
    failure_code text,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id,id),
    FOREIGN KEY (organization_id,product_id) REFERENCES products(organization_id,id)
);
CREATE INDEX product_images_reservations ON product_images(organization_id,uploaded_by,created_at);
ALTER TABLE products ADD COLUMN image_id uuid;
ALTER TABLE products ADD COLUMN image_key text;
ALTER TABLE products ADD COLUMN image_status text CHECK (image_status IN ('PROCESSING','READY','FAILED'));
ALTER TABLE products ADD CONSTRAINT products_image_tenant_fk
    FOREIGN KEY (organization_id,image_id) REFERENCES product_images(organization_id,id);
CREATE INDEX products_image_id ON products(image_id) WHERE image_id IS NOT NULL;
