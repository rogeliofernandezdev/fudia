ALTER TABLE roles
  DROP CONSTRAINT IF EXISTS roles_permissions_no_wildcard,
  DROP CONSTRAINT IF EXISTS roles_menu_access_no_wildcard;

DROP INDEX IF EXISTS users_single_platform_admin_uq;

UPDATE roles
SET name='Administrador',
    description='Acceso administrativo a la empresa y sus locales',
    updated_at=now()
WHERE system_key='administrator';
