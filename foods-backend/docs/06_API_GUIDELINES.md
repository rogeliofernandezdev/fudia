# API

`POSOrderDetail.receiptContext` expone identidad actual de empresa/local,
dirección/teléfono y moneda/zona del perfil fiscal del local, con el mismo permiso
`orders.read` y alcance de sesión. Permite un ticket operativo de pago de 80 mm;
no emite un comprobante fiscal ni genera series/SUNAT. Leer el detalle no registra
pagos. No requiere migración ni amplía permisos.

REST JSON descrito en api/openapi.yaml. Los errores usan código estable, mensaje
seguro y correlationId. Paginación, filtros, idempotencia y concurrencia se
definen en OpenAPI antes del frontend.

## Registro por lote de mesas

`POST /v1/admin/tables/batch` exige `tables.manage` y sesión de empresa/local.
Acepta hasta 500 filas y conserva el orden de entrada en la respuesta. Valida
zonas activas bajo bloqueo compartido, inserta todo el lote y registra una sola
auditoría con los UUID creados en una única sentencia SQL transaccional. No hay
consultas por fila ni auditoría después del guardado. Cualquier conflicto o fallo
de auditoría revierte todo el lote. Los UUID y tokens QR se generan en PostgreSQL.
No requiere migración ni cambia los permisos existentes.

La API cancela el procesamiento a los 30 s y conserva 35 s para responder; el
BFF espera hasta 40 s. La conexión PostgreSQL usa 5 s como límite por defecto
si `connect_timeout` no está configurado. Estos límites no garantizan respuesta
si se corta la red después del commit: el cliente no reintenta escrituras y
presenta el resultado como no confirmado, conservando borrador y refrescando
el listado antes de una posible repetición.

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

## Responsable de la atención en mesa

Crear una comanda manual de Salón asigna `orders.waiter_id` al usuario autenticado de la empresa;
el cliente no puede elegirlo ni sustituirlo en una edición. `Order` expone
`waiterId`, `waiterName` y `collectedByNames` en Pedidos, Salón y detalle de POS.
Los nombres de cobro proceden exclusivamente de `payments.created_by`; no se
atribuyen al abridor de caja ni al mozo. Un pago dividido conserva todos sus autores.
Las proyecciones SQL respetan empresa/local y no hacen peticiones por cada fila.

Editar, cancelar, enviar a cocina o confirmar la entrega de una mesa asignada
exige `orders.manage` y ser su mozo. Otro usuario recibe 403
`order_assigned_to_another_waiter`, incluso con un permiso amplio. La propiedad
se verifica bajo el mismo bloqueo de pedido que protege la mutación. La lectura
sigue disponible con `orders.read`. Cocina conserva `kitchen.manage` para preparar
y POS conserva `cash.manage` y la asignación de turno para cobrar: no cambian el mozo.
Los pedidos sin asignación conservan sus permisos anteriores; no se asigna un mozo
ficticio a WhatsApp. La asignación pertenece al pedido/servicio, no a la mesa permanente.

Un administrador de plataforma que opera un tenant ajeno no se convierte en mozo
de esa empresa: no se persiste una asignación que cruce la frontera del tenant.

La migración 000072 agrega una FK por empresa y vincula las comandas históricas
manuales de Salón a su autor original cuando pertenece a la misma empresa.

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

`PATCH /v1/admin/product-availability/{productId}` (y su alias de Operación)
exige `menu.manage` y `reason` no vacío de hasta 240 caracteres para cambios
manuales de cupo o Agotado/Reactivar. Estado y cantidad conservan sus invariantes;
el motivo no sustituye `note`. Guarda valores anteriores/nuevos, porciones
vendidas al cambiar, fecha operativa, autor y motivo en `audit_log` dentro de
la misma transacción: si falla auditoría no cambia disponibilidad. Un reintento
con el mismo estado/cupo/nota no crea otro cambio. La hora usa `clock_timestamp()`
después de bloquear el producto para ordenar cambios concurrentes correctamente.

`GET /v1/admin/product-availability/{productId}/history` y el alias de Operación
exigen `menu.read`, filtran empresa/local de sesión y producto y paginan todos
los días en orden descendente estable. Conteo y filas comparten snapshot.
Los eventos antiguos sin metadata devuelven null, sin inventar valores/motivos.
La migración 78 añade solo un índice parcial de lectura; no altera cantidades.

`POST /v1/admin/products` exige `menu.manage` y `initialPortionQuantity` entera
positiva cuando `quantityControl=portions`. Sin control e Inventario no aceptan
ese campo. Producto, porciones de hoy del local de sesión y auditoría se confirman
en una transacción; cualquier fallo revierte todos los registros. La fecha se
deriva de la zona del local en PostgreSQL, nunca del navegador. PATCH comercial
no admite cantidad inicial ni modifica la cantidad diaria; Disponibilidad conserva
los ajustes y el control de Agotado/Reactivar. No requiere migración.

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

`GET /v1/admin/cash-shifts/{id}/report` requiere `cash.read` y alcance de empresa
y local. Devuelve identidad, turno, productos de pedidos con cobros, cada pago
con medio/autor/referencia, movimientos con motivo/autor, denominaciones y
totales monetarios decimales. `EXISTS` evita duplicar productos por pagos
divididos. Los valores de productos son del pedido completo: no se prorratean
como cobros del turno. Reversos se atribuyen al turno donde se registraron.
Los turnos abiertos se leen en una instantánea consistente; los cerrados nuevos
devuelven la copia persistida. Los cierres antiguos se reconstruyen y se
identifican mediante `persisted=false`, sin inventar una instantánea histórica.
En cierre ciego, sin `cash.expected.read`, el informe abierto responde 403 y
solo queda disponible después del cierre. Ninguna consulta registra movimientos.
El cierre conserva su respuesta `CashShift`, pero guarda el informe antes de
confirmar la transacción. Fallar en el informe revierte también el cierre.
Despliegue: aplicar migración 79 y actualizar/recrear el backend; ejecutar solo
migraciones no incorpora rutas nuevas a una imagen antigua.

Los timestamps `CashShift.openedAt`, `CashShift.closedAt` y
`CashMovement.createdAt` se serializan en RFC 3339 UTC con sufijo `Z` y precisión
de microsegundos, independientemente de la zona horaria de la sesión PostgreSQL.
`closedAt` permanece null mientras el turno esté abierto. `businessDate` es una
fecha de operación distinta y no reemplaza apertura ni cierre. Las lecturas de
cajas, turno actual, historial y detalle usan la misma proyección. No cambia
el instante almacenado ni se necesita una migración.

## Cuenta de mesa y destinos de atención

`GET /v1/admin/products` devuelve `ProductListItem`: añade `availableQuantity`
e `inventoryUnit` a los datos comerciales. Se calcula en una consulta paginada,
por empresa y local de sesión, con la fecha local para las porciones. No cuenta
cupos de otros días ni existencias de otros locales. Sin control devuelve null;
agotado, inactivo o fuera de horario devuelve cero para controles con cantidad.
La descripción permanece en el contrato para editar sin perder información.

- `GET /v1/admin/service-destinations` entrega el catálogo de destinos; Productos también devuelve `serviceDestinationOptions`.
- `POST /v1/admin/orders/{id}/items` agrega solo consumo nuevo a cuenta Abierta con `requestKey`, cantidades y elecciones. Precios, cupos e inventario se validan en backend. Cuenta cerrada devuelve `409 order_bill_closed`; los reintentos de rondas ya confirmadas conservan idempotencia.
- `POST /v1/admin/orders/{id}/bill/close` exige todos los productos entregados y pasa la cuenta a Por cobrar sin liberar la mesa. Productos pendientes o agregado distinto de Entregado devuelven `409 products_not_delivered`. Cobros individual y dividido de Salón exigen este cierre y verifican de nuevo cada producto bajo bloqueo del pedido.
- `PATCH /v1/admin/orders/{id}/service-items/{itemId}/deliver` entrega una línea lista. Las tres mutaciones exigen `orders.manage`, tenant/local y mozo asignado.
- KDS admite `destination=kitchen|bar`; no devuelve Entrega directa. El identificador de ticket es opaco y separa ronda/estación. Preparar Cocina exige `kitchen.manage`; Barra exige `bar.manage`.
- «Encargado de barra» tiene `menu.read`, `orders.read`, `bar.manage` y acceso a Cocina/Barra. Solo Administrador de empresa incorpora automáticamente `bar.manage`; otros roles existentes conservan sus permisos. No se asigna el rol a usuarios automáticamente.
- Pagos individual/dividido, cierre y entrega bloquean la misma fila. Liberación automática exige cierre + pago completo + todos entregados. Una cuenta pagada no se amplía ni reabre.
- Concierge conserva idempotencia por solicitud/ronda y puede ampliar la cuenta de Salón solo mientras Abierta, sin reenviar consumos anteriores. Después del cierre devuelve `409 order_bill_closed`, igual que el flujo del mozo.
