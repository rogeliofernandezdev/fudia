-- Separate register configuration from operating a shift. Never derive this
-- administrative capability from cash.manage, a role label or other permissions.
UPDATE roles
SET permissions=array_append(permissions,'cash.registers.manage')
WHERE system_key='administrator'
  AND NOT ('cash.registers.manage'=ANY(permissions));
