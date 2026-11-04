-- Campana de avisos (GET /v1/admin/notices, cada 15 s por sesión): el filtro de
-- pendientes busca, por cada aviso de pedido nuevo, si ya existe su aviso «listo».
-- Sin un índice por pedido esa búsqueda recorría el índice de toda la empresa en
-- cada fila: 4,6 s en caliente con un año de historia; con este índice, ~20 ms.
CREATE INDEX IF NOT EXISTS staff_notices_order_kind_idx
  ON staff_notices(organization_id, order_id, kind, created_at)
  WHERE order_id IS NOT NULL;

-- Listado de Pedidos: el orden «abiertos primero, luego por fecha» usaba un
-- recorrido completo del local y un top-N sort (325 ms con 200 mil pedidos). La
-- expresión coincide literalmente con el ORDER BY de orders/infrastructure/postgres/read.go;
-- si cambia ese orden, este índice debe cambiar con él.
CREATE INDEX IF NOT EXISTS orders_list_order_idx
  ON orders(
    organization_id,
    location_id,
    (CASE WHEN status='cancelado' OR completed_at IS NOT NULL OR (status='entregado' AND channel<>'salon') THEN 1 ELSE 0 END),
    created_at DESC
  );
