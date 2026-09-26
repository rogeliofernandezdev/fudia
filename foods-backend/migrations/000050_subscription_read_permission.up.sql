UPDATE roles
SET permissions=array_append(permissions,'subscription.read'),updated_at=now()
WHERE system_key='administrator'
  AND NOT ('subscription.read'=ANY(permissions));
