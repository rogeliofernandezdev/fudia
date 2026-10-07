# API

REST JSON descrito en api/openapi.yaml. Los errores usan código estable, mensaje
seguro y correlationId. Paginación, filtros, idempotencia y concurrencia se
definen en OpenAPI antes del frontend.

## Resumen administrativo del restaurante

`GET /v1/admin/dashboard` exige `dashboard.read` y alcance de empresa/local de
la sesión. Los agregados se leen en una transacción de solo lectura con snapshot
repetible: indicadores, gráfico, productos y `operations` describen la misma
instantánea. Cualquier error de lectura responde 503, nunca listas vacías falsas.
`businessDate` usa la zona horaria del local; cobros y reservas de hoy usan esa fecha.
`paidOrders` cuenta pedidos no cancelados totalmente pagados con al menos un pago
hoy, no cada pedido con un abono. `averageTicket` promedia los totales de esos pedidos.
Se conserva la integridad de `salesNet` ante ajustes/devoluciones existentes.
`operations` consolida saldos de pedidos no cancelados ni finalizados (también
anteriores), pagos parciales, mesas activas ocupadas, colas de cocina y delivery,
cajas activas, turnos abiertos y productos agotados. El efectivo esperado suma
aperturas y movimientos de efectivo de los turnos abiertos, no pagos con tarjeta.
Si hay una caja ciega y el usuario no tiene `cash.expected.read` ni privilegio
administrador, `cashBalance` es null. La disponibilidad respeta cupos por día,
inventario, control manual, horarios y opciones requeridas de menús/combos.
No requiere nuevas tablas ni migraciones. El panel consolida el local seleccionado,
no suma locales ni monedas distintas.

El historial de Ventas consume `GET /v1/admin/pos/orders?paymentStatus=paid` y
abre su detalle mediante `GET /v1/admin/pos/orders/{id}`, con `orders.read` y
alcance de empresa/local. Devuelve productos, subtotal, delivery, total y pagos
netos de devoluciones. Cada pago incluye `methodName` desde el catálogo de su
empresa, incluso si el medio quedó inactivo; no se reconstruyen nombres en cliente.
El listado incluye `paymentMethods`: nombres únicos ordenados del catálogo de la
empresa para pagos con importe neto positivo. Incluye medios ahora inactivos,
excluye cobros totalmente devueltos y devuelve `[]` cuando no hay cobros vigentes.
El agregado SQL respeta empresa/local y no exige consultas adicionales por fila;
pagos divididos conservan todos sus medios en una única venta paginada.

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

- `GET /v1/admin/pos/orders` admite `from` y `to` sobre `orders.created_at`,
  consistente con la fecha visible de Ventas. Aplica el mismo rango al conteo y
  al listado paginado; admite límites independientes y rechaza rangos invertidos
  con 400 `invalid_order_date_filter`. Sin fechas conserva el historial completo.
- Los filtros `from` y `to` reciben fechas ISO (`YYYY-MM-DD`) y se validan; un
  valor con otro formato responde 400.
- Cuando el campo filtrado es un `timestamptz` (`created_at`), los límites se
  interpretan en la zona horaria del local activo: `from` es la medianoche local
  de ese día y `to` es exclusivo hasta la medianoche local del día siguiente.
  Nunca se compara un `timestamptz` directamente contra `::date`, porque eso
  aplica la zona de la sesión de PostgreSQL (UTC) y desplaza los días.
- Los campos que ya son fechas de negocio (`business_date`) se comparan
  directamente.

## Fechas auditables de Caja

Los timestamps `CashShift.openedAt`, `CashShift.closedAt` y
`CashMovement.createdAt` se serializan en RFC 3339 UTC con sufijo `Z` y precisión
de microsegundos, independientemente de la zona horaria de la sesión PostgreSQL.
`closedAt` permanece null mientras el turno esté abierto. `businessDate` es una
fecha de operación distinta y no reemplaza apertura ni cierre. Las lecturas de
cajas, turno actual, historial y detalle usan la misma proyección. No cambia
el instante almacenado ni se necesita una migración.
