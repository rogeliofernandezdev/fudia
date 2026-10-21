ALTER TABLE products ADD COLUMN service_destination text NOT NULL DEFAULT 'kitchen'
  CHECK (service_destination IN ('kitchen','bar','direct'));
-- Existing prepared products keep their current route. Only explicit retail
-- classification is mapped to direct service; names and categories are not used.
UPDATE products SET service_destination='direct' WHERE product_type='retail';
ALTER TABLE orders ADD COLUMN bill_closed_at timestamptz;
ALTER TABLE orders ADD COLUMN bill_closed_by uuid;
ALTER TABLE orders ADD CONSTRAINT orders_bill_closed_by_tenant_fk
  FOREIGN KEY (bill_closed_by,organization_id) REFERENCES users(id,organization_id);
UPDATE orders SET bill_closed_at=COALESCE(completed_at,now())
  WHERE completed_at IS NOT NULL OR EXISTS(SELECT 1 FROM payments p WHERE p.order_id=orders.id);
ALTER TABLE order_kitchen_rounds ADD COLUMN request_key text;
CREATE UNIQUE INDEX order_round_request_key_idx ON order_kitchen_rounds(organization_id,order_id,request_key)
  WHERE request_key IS NOT NULL;

CREATE TABLE order_service_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id uuid NOT NULL,
  order_id uuid NOT NULL,
  order_item_id uuid NOT NULL,
  product_id uuid,
  name text NOT NULL,
  qty numeric(10,2) NOT NULL CHECK(qty>0),
  destination text NOT NULL CHECK(destination IN ('kitchen','bar','direct')),
  status text NOT NULL CHECK(status IN ('nuevo','confirmado','preparando','listo','entregado')),
  delivered_by uuid,
  delivered_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (order_id,organization_id) REFERENCES orders(id,organization_id) ON DELETE CASCADE,
  FOREIGN KEY (order_item_id,organization_id) REFERENCES order_items(id,organization_id) ON DELETE CASCADE,
  FOREIGN KEY (product_id,organization_id) REFERENCES products(id,organization_id),
  FOREIGN KEY (delivered_by,organization_id) REFERENCES users(id,organization_id)
);
CREATE INDEX order_service_items_scope_idx ON order_service_items(organization_id,order_id,status,destination);
INSERT INTO order_service_items(organization_id,order_id,order_item_id,product_id,name,qty,destination,status)
SELECT i.organization_id,i.order_id,i.id,i.product_id,i.name,i.qty,COALESCE(p.service_destination,'kitchen'),
 CASE WHEN o.status='entregado' THEN 'entregado' WHEN o.status='nuevo' THEN 'nuevo'
      WHEN p.service_destination='direct' THEN 'listo'
      ELSE COALESCE(r.status,CASE WHEN o.status IN ('preparando','listo') THEN o.status ELSE 'confirmado' END) END
FROM order_items i JOIN orders o ON o.id=i.order_id AND o.organization_id=i.organization_id
LEFT JOIN products p ON p.id=i.product_id AND p.organization_id=i.organization_id
LEFT JOIN order_kitchen_rounds r ON r.id=i.kitchen_round_id AND r.organization_id=i.organization_id
WHERE NOT EXISTS(SELECT 1 FROM order_item_combo_selections s WHERE s.order_item_id=i.id AND s.organization_id=i.organization_id);
INSERT INTO order_service_items(organization_id,order_id,order_item_id,product_id,name,qty,destination,status)
SELECT i.organization_id,i.order_id,i.id,s.option_product_id,s.option_name,i.qty,p.service_destination,
 CASE WHEN o.status='entregado' THEN 'entregado' WHEN o.status='nuevo' THEN 'nuevo'
      WHEN p.service_destination='direct' THEN 'listo'
      ELSE COALESCE(r.status,CASE WHEN o.status IN ('preparando','listo') THEN o.status ELSE 'confirmado' END) END
FROM order_item_combo_selections s JOIN order_items i ON i.id=s.order_item_id AND i.organization_id=s.organization_id
JOIN orders o ON o.id=i.order_id AND o.organization_id=i.organization_id
JOIN products p ON p.id=s.option_product_id AND p.organization_id=s.organization_id
LEFT JOIN order_kitchen_rounds r ON r.id=i.kitchen_round_id AND r.organization_id=i.organization_id;

-- Authorized: only the company administrator gets the new action automatically.
UPDATE roles SET permissions=array_append(permissions,'bar.manage')
 WHERE system_key='administrator' AND NOT ('bar.manage'=ANY(permissions));
INSERT INTO roles(organization_id,name,system_key,description,menu_access,permissions,active)
SELECT id,'Encargado de barra','bartender','Prepara las bebidas de Barra',ARRAY['cocina'],ARRAY['menu.read','orders.read','bar.manage'],true
FROM organizations ON CONFLICT(organization_id,name) DO NOTHING;

