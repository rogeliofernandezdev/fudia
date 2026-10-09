ALTER TABLE orders ADD COLUMN waiter_id uuid;
ALTER TABLE orders ADD CONSTRAINT orders_waiter_tenant_fk
  FOREIGN KEY (waiter_id, organization_id) REFERENCES users(id, organization_id);

-- Manual table orders already record the person who took the first comanda.
-- WhatsApp orders have a different channel and must not acquire a fictitious waiter.
UPDATE orders o SET waiter_id=o.created_by
FROM users u
WHERE o.channel='salon' AND o.table_id IS NOT NULL
  AND u.id=o.created_by AND u.organization_id=o.organization_id;
