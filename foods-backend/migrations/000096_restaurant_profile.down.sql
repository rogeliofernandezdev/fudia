DROP TABLE organization_menu_images;
ALTER TABLE locations DROP COLUMN opened_on;
ALTER TABLE organizations DROP CONSTRAINT organization_logo_image_fk;
ALTER TABLE organizations DROP COLUMN logo_image_id;
ALTER TABLE organizations DROP COLUMN founded_on;
ALTER TABLE product_images DROP CONSTRAINT media_product_purpose;
ALTER TABLE product_images DROP COLUMN purpose;
