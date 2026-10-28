UPDATE roles
SET permissions=array_remove(permissions,'cash.registers.manage')
WHERE 'cash.registers.manage'=ANY(permissions);
