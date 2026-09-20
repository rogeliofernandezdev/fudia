ALTER TABLE users ADD COLUMN platform_admin boolean NOT NULL DEFAULT false;

UPDATE users SET platform_admin=true WHERE lower(email)='admin@foods.local';
