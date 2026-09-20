DROP INDEX IF EXISTS user_roles_role_idx;
DROP INDEX IF EXISTS roles_system_key_uq;
ALTER TABLE roles DROP COLUMN IF EXISTS updated_at, DROP COLUMN IF EXISTS created_at, DROP COLUMN IF EXISTS active, DROP COLUMN IF EXISTS description, DROP COLUMN IF EXISTS system_key;
