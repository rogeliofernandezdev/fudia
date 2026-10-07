# API

REST JSON descrito en api/openapi.yaml. Los errores usan código estable, mensaje
seguro y correlationId. Paginación, filtros, idempotencia y concurrencia se
definen en OpenAPI antes del frontend.

El historial de Ventas consume `GET /v1/admin/pos/orders?paymentStatus=paid` y
abre su detalle mediante `GET /v1/admin/pos/orders/{id}`, con `orders.read` y
alcance de empresa/local. Devuelve productos, subtotal, delivery, total y pagos
netos de devoluciones. Cada pago incluye `methodName` desde el catálogo de su
empresa, incluso si el medio quedó inactivo; no se reconstruyen nombres en cliente.

## Alta manual de delivery

`POST /v1/admin/orders` conserva `orders.manage` y el alcance de empresa/local
derivado de la sesión. Delivery requiere nombre, teléfono válido de 7 a 15 dígitos
y dirección; el número admite `+` inicial y separadores habituales. El envío es
no negativo y no puede asociarse una mesa. `sendToKitchen=true` registra el estado
Confirmado para la cola existente de Cocina. Precios, composición y disponibilidad
se revalidan en backend; el cliente no necesita enviar precios ni nombres comerciales.
No se requieren tablas ni migraciones nuevas para esta capacidad. Despacho y cobro
conservan sus endpoints, permisos y transacciones actuales; la logística avanzada
de repartidores, zonas o seguimiento geográfico sigue fuera de este alcance.

## Lookups de catálogo

Los endpoints usados por autocompletes exponen `q`, `page` y `pageSize` y devuelven `items`, `total`, `page` y `pageSize`. El backend nunca obliga al frontend a descargar el catálogo completo: la apertura puede pedir una página pequeña y las búsquedas posteriores recorren la paginación según necesidad.

## Filtros por fecha

- Los filtros `from` y `to` reciben fechas ISO (`YYYY-MM-DD`) y se validan; un
  valor con otro formato responde 400.
- Cuando el campo filtrado es un `timestamptz` (`created_at`), los límites se
  interpretan en la zona horaria del local activo: `from` es la medianoche local
  de ese día y `to` es exclusivo hasta la medianoche local del día siguiente.
  Nunca se compara un `timestamptz` directamente contra `::date`, porque eso
  aplica la zona de la sesión de PostgreSQL (UTC) y desplaza los días.
- Los campos que ya son fechas de negocio (`business_date`) se comparan
  directamente.
