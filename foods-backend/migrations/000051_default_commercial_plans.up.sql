INSERT INTO subscription_plans(
  code,name,description,currency,monthly_price,annual_price,trial_days,
  max_locations,max_users,module_keys,terms_version,active
) VALUES
(
  'emprende','Emprende',
  'Para restaurantes que empiezan a digitalizar su operación con un solo local.',
  'PEN',89.00,890.00,14,1,6,
  ARRAY['reportes','pos','pedidos','cocina','mesas','caja','productos','clientes','locales','fiscal','usuarios']::text[],
  '2026-09',true
),
(
  'impulso','Impulso',
  'Para negocios en crecimiento que necesitan inventario, recetas, reservas y más control.',
  'PEN',169.00,1690.00,14,3,20,
  ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','clientes','locales','fiscal','usuarios']::text[],
  '2026-09',true
),
(
  'escala','Escala',
  'Para operaciones multi-local que requieren abastecimiento y control integral.',
  'PEN',299.00,2990.00,14,10,60,
  ARRAY['reportes','pos','pedidos','cocina','mesas','caja','reservas','productos','combos','recetas','inventario','kardex','compras','clientes','locales','fiscal','usuarios']::text[],
  '2026-09',true
)
ON CONFLICT(code) DO NOTHING;
