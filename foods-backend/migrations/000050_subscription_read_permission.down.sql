UPDATE roles
SET permissions=array_remove(permissions,'subscription.read'),updated_at=now()
WHERE system_key='administrator';
