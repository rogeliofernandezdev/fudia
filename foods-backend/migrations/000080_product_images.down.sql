ALTER TABLE products DROP CONSTRAINT products_image_tenant_fk;
ALTER TABLE products DROP COLUMN image_id, DROP COLUMN image_key, DROP COLUMN image_status;
DROP TABLE product_images;
