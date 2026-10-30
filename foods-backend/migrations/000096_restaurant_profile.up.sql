ALTER TABLE product_images ADD COLUMN purpose text NOT NULL DEFAULT 'product'
    CHECK (purpose IN ('product','restaurant_logo','virtual_menu'));
ALTER TABLE product_images ADD CONSTRAINT media_product_purpose CHECK (purpose='product' OR product_id IS NULL);
ALTER TABLE organizations ADD COLUMN founded_on date;
ALTER TABLE organizations ADD COLUMN logo_image_id uuid;
ALTER TABLE organizations ADD CONSTRAINT organization_logo_image_fk
    FOREIGN KEY(id,logo_image_id) REFERENCES product_images(organization_id,id);
ALTER TABLE locations ADD COLUMN opened_on date;
CREATE TABLE organization_menu_images (
    organization_id uuid NOT NULL REFERENCES organizations(id),
    image_id uuid NOT NULL,
    sort_order integer NOT NULL CHECK (sort_order>=0),
    PRIMARY KEY(organization_id,image_id),
    UNIQUE(organization_id,sort_order),
    FOREIGN KEY(organization_id,image_id) REFERENCES product_images(organization_id,id)
);
