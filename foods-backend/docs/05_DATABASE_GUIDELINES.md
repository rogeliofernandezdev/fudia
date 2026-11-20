# Base de datos

PostgreSQL es la fuente de verdad. Las migraciones son versionadas y se aplican
hacia adelante. Las claves foráneas son UUID sin autogeneración.

Pedido, stock, pago y emisión usan transacciones explícitas. No se modifica stock
o saldo sin documento y movimiento asociado.

## Performance

Los endpoints paginados deben evitar N+1 y consultas por fila. Antes de agregar
caché, se reduce el número de round trips y se agrupan cálculos en SQL cuando sea
seguro.

Con la base remota cada consulta cuesta una ida y vuelta de red (~150-230 ms
medidos desde fuera de la región). Por eso:

- Cada petición autenticada hace una sola consulta de sesión y permisos.
- Al crear o editar un pedido, los productos se precargan en una consulta
  (`OrderProducts`: combo, precio, disponibilidad y si tiene modificadores); los
  modificadores solo se consultan si el producto los tiene o el pedido los envía.
  Un producto que la precarga no devuelve se valida con las consultas
  individuales anteriores, conservando sus mensajes.
- El consumo de inventario bloquea todos los productos del pedido en una consulta
  ordenada por id (`LockQuantityProducts`, mismo orden de bloqueo que el ciclo
  anterior) y busca sus recetas activas en otra (`ActiveRecipes`).
- `InsertItems` inserta cada línea con sus selecciones de combo, modificadores y
  líneas de servicio en una sentencia, y envía todas las líneas en un `pgx.Batch`.
- Las lecturas compuestas también van en un `pgx.Batch` (una ida y vuelta; fuera de
  transacción corre en una transacción implícita, así que todo sale de la misma
  instantánea). `Items` encola líneas, combos y modificadores (antes 3 consultas);
  el detalle del pedido (`Detail`, que Salón y Pedidos refrescan cada 10 s) encola
  pedido, entregas, líneas y cobrado neto (antes 6 consultas secuenciales); Cocina
  lee las líneas y el tiempo objetivo de todas sus comandas con `TicketContents`
  en un solo batch, en vez de cuatro consultas por comanda. La campana
  (`GET /v1/admin/notices`, cada 15 s) lee el contador y la lista en un batch.

Prueba de carga (100 restaurantes con Cocina, Salón y Pedidos refrescando cada
10 s, POS y Caja cada 30 s, y el ciclo completo de mesa) sobre Postgres
desechable: con la API en la misma región que la base soporta 100 restaurantes,
incluso a 6 veces el ritmo de hora punta, sin errores y con p95 menor a 160 ms.
La base remota acepta como máximo 40 conexiones, por lo que la API debe
desplegarse en su misma región; `DATABASE_MAX_CONNS` no puede compensar la
latencia.

Las búsquedas `ILIKE '%texto%'` de catálogos operativos usan `pg_trgm` e índices
GIN sobre los campos realmente consultados. Los filtros por tenant/local/estado
usan índices B-tree o parciales acordes al patrón de lectura.

`pageSize` permanece acotado. La paginación por offset es válida para los
listados administrativos actuales; cuando una tabla crezca hasta volver costosos
los offsets altos, se migra ese endpoint a keyset/cursor conservando un orden
estable.

Las consultas identificadas como lentas deben revisarse con `EXPLAIN (ANALYZE,
BUFFERS)` sobre un volumen representativo antes de introducir índices adicionales.

Los filtros por día del negocio comparan la columna con límites calculados en la
zona horaria del local (`created_at >= inicio_del_día_local`), nunca con
`(created_at AT TIME ZONE tz)::date`: la conversión por fila impide usar los
índices `(organization_id, location_id, created_at)` y obliga a leer todo el
historial. Para pedidos pagados en un periodo, el conjunto se arma desde los cobros
del periodo (`o.id IN (SELECT order_id FROM payments …)`), no recorriendo todos los
pedidos del local. En reporting, `localDayStartSQL` centraliza ese límite.

Los reportes que comparten el pool con la operación (dashboard) corren en una
transacción de solo lectura, que no bloquea escrituras, y fijan
`SET LOCAL statement_timeout` (5 s) para que una consulta degradada libere la
conexión en vez de retenerla. Los pedidos en curso usan el índice parcial
`orders_active_unsettled_idx` (`status<>'cancelado' AND completed_at IS NULL`).
Revisión de consultas calientes (2026-11-04), con `EXPLAIN (ANALYZE, BUFFERS)` en
caliente sobre PostgreSQL 16 desechable con un local de un año (200 mil pedidos,
1,4 M líneas, 1,1 M avisos) y diez empresas de relleno:

- Campana: el filtro `pending` buscaba el aviso «listo» de cada pedido sin índice
  por pedido y recorría el índice de toda la empresa por fila (4,6 s, 475 mil
  buffers). `staff_notices_order_kind_idx` (migración 000109) lo deja en ~20 ms; la
  subconsulta de rondas filtra también por local para usar su índice único.
- Listado de Pedidos: el `ORDER BY` «abiertos primero» obligaba a leer el local
  completo (325 ms). `orders_list_order_idx` repite literalmente esa expresión
  (`listOrderSQL`, protegido por `TestListOrderMatchesIndex`): 1,7 ms en la página
  1 y 3 ms en la 50, también con plan genérico.
- Sin cambios porque ya usan índices: piso de Salón (3 ms), tickets de Cocina
  (`orders_active_unsettled_idx`, ~1 200 buffers), cobrado neto por pedido, cobros por
  turno y la sesión autenticada (`sessions_lookup_idx`).
- Listado de POS (polling cada 30 s en POS y Caja): el filtro «por cobrar» y su
  total calculaban el cobrado neto con dos subconsultas por cada pedido no cancelado
  del local (5,6 s, 1,2 M buffers). Desde la migración 000110 el pedido guarda ese
  valor en `orders.paid_amount` y POS filtra y totaliza con él usando el índice
  parcial `orders_pos_unpaid_idx`: el total baja a 20-65 ms y la página a 5-14 ms
  (plan genérico incluido). Una reescritura sin desnormalizar (agregado por hash)
  solo llegaba a 1,3-1,9 s y empeoraba los filtros por fecha. No se puede suponer
  «completado ⇒ pagado» para acotar la búsqueda: la migración 70 marcó como
  completados pedidos entregados sin cobro.

- Detalle de menú para la comanda (`OrderCombo`): las opciones de todos los grupos
  se leen en una consulta (antes, una por grupo), y el cupo diario de cada opción
  compara `created_at` con los límites del día local en vez de convertir cada fila.
  Con 231 mil selecciones históricas: 341 ms por grupo → 58 ms para todos los
  grupos. El mismo cambio de límites se aplicó al listado de menús de la comanda,
  a la carta pública/Concierge (`public_menu.go`) y a la validación de cupo al crear
  pedidos (`ExternalOptionUsage`).
- `GET /v1/admin/order-combo-details?ids=` devuelve hasta 20 menús en una petición
  con las mismas reglas que `GET /order-combos/{id}`; la comanda lo usa para refrescar
  los menús elegidos (antes, una petición y una consulta de sesión por menú).
- `GET /v1/admin/products?productIds=` acota el listado a un lote de hasta 100
  productos con la misma cantidad disponible; el editor de menús refresca así solo
  las opciones elegidas en vez de descargar toda la carta cada 10 s.

Revisión de rendimiento (2026-11-20), medida con `EXPLAIN (ANALYZE, BUFFERS)` en
PostgreSQL 16 desechable con un local de un año (200 mil pedidos, 600 mil líneas,
400 mil selecciones de menú, 1,2 M movimientos de Kárdex, 1,4 M filas de
auditoría) y diez empresas de relleno:

- Migración 000137 (`IF NOT EXISTS`; en producción con tablas grandes, crearlos
  antes con `CONCURRENTLY`):
  - `orders_location_created_idx (organization_id, location_id, created_at DESC, id DESC)`:
    ningún índice de pedidos servía un rango u orden por fecha sin filtrar estado.
    Página de POS/Ventas 60 ms → 0,1 ms; total del día 5 ms → 1,5 ms. El total sin
    filtro de fechas sigue recorriendo el historial del local (~60 ms).
  - `audit_log_concierge_order_idx` (parcial, `concierge.order.created`): el pedido
    QR consultaba la auditoría con el pedido bloqueado, 60 ms → 0,1 ms.
  - `stock_movements_location_created_idx` y `stock_movements_item_created_idx`:
    Kárdex 110 ms y 43 ms → < 0,1 ms. El filtro por artículo compara `uuid`, no
    texto.
  - `delivery_dispatches_due_idx` (parcial): el barrido de delivery dejaba de
    recorrer todo el historial de despachos (15 ms → 1 ms por barrido).
  - `customers_org_updated_idx`: listado de clientes 14 ms → < 0,1 ms.
- Comparar ids recibidos como texto con `col=$n::uuid` usando
  `database.UUIDArg` (vacío → NULL para filtros opcionales, formato inválido → un
  uuid que no existe), nunca `col::text=$n`: convertir la columna impide el índice.
  El seguimiento del pedido del chat web (cada 5 s) dejó de recorrer los pedidos
  del local (39 ms → 0,07 ms).
- Cupo diario de opciones de menú al crear pedidos (`ExternalOptionUsage`): los
  límites del día local salen de un CTE de una fila para que el planificador
  recorra solo los pedidos de hoy (27 ms → 2,4 ms, también con plan genérico). Las
  subconsultas correlacionadas (`OrderCombos`, carta pública) ya usan el índice
  nuevo porque reciben la zona horaria como parámetro externo.
- Las búsquedas de devoluciones y rondas de cocina filtran también por local
  (`pr.location_id=p.location_id`, `k.location_id=o.location_id`) para usar sus
  índices `(organization_id, location_id, …)`.
- Consumo de inventario por lotes: porciones del día, artículos de inventario,
  saldos (`LockBalances`) e insumos de todas las recetas se leen y bloquean en una
  consulta por grupo, en orden de id; porciones, saldos y Kárdex se guardan juntos
  con `ApplyStockWrites` (un `pgx.Batch`). Antes eran unas cuatro idas y vueltas
  por insumo con los saldos bloqueados. Los mensajes y el orden de validación no
  cambian.
- Lecturas compuestas en un `pgx.Batch`: dashboard (≈17 consultas → 1 ida y
  vuelta, dentro de su transacción de solo lectura con `SET LOCAL
  statement_timeout`), listado de Pedidos (5 → 1) y Cocina (`KitchenQueues`, 3 → 1).
- Turnos de caja: ingresos, egresos y movimientos salen de un `LEFT JOIN LATERAL`
  (`cashShiftFrom`) en vez de cinco subconsultas por turno.
- Disponibilidad de menús (`BlockedCombos`) y opciones de grupos (carta pública y
  editor de menús) en una consulta por página o menú, no una por menú o grupo.
- Al preparar un pedido, un menú repetido en varias líneas se lee una sola vez.
- Cupos diarios de opciones de menú al crear o editar un pedido: todas las
  opciones se bloquean en una consulta en orden de id (`LockOptionQuotas`) y lo ya
  pedido hoy, dentro y fuera del pedido, se lee después en otra (`OptionUsages`),
  así la lectura ve lo confirmado por el pedido que tenía el bloqueo. Antes eran dos
  o tres consultas por opción. `TestComboOptionDailyQuotaAcrossOrdersAndEdits`
  cubre otros pedidos del día, edición y mensajes.
- Ventas/POS leen la zona horaria del local con una subconsulta escalar en vez de
  unir `locations`: el JOIN impedía recorrer los pedidos en el orden del índice y
  una página profunda de Ventas sin rango ordenaba todo el historial pagado
  (70-85 ms → 1,5-7 ms). El total de Ventas sin rango de fechas sigue sumando todo
  el historial pagado (~45 ms con un año); solo un resumen precalculado lo
  evitaría. POS siempre filtra «por cobrar» con `orders_pos_unpaid_idx` (0,1 ms).
- Paginación por offset medida en páginas profundas: Pedidos página 500 1,8 ms,
  Kárdex página 2 500 7,7 ms, Ventas página 500 ≤ 7 ms. No hace falta cursor todavía.
- Dashboard: los pedidos pagados del periodo se calculan una vez (CTE
  `MATERIALIZED`) para el promedio, lo más vendido, los pedidos por canal y las
  categorías, y los cobros del periodo y del anterior (`movements`) una vez para la
  comparación, la tendencia y los desgloses por canal y medio de pago; cada grupo
  sale como JSON de una sola sentencia. «Pedido pagado» usa `orders.paid_amount`.
  Carga completa con un año de datos: 1 día 139 → 56 ms, 7 días 337 → 110 ms,
  30 días 1 027 → 404 ms.
- El API registra `db_pool_saturated` (esperas, adquisiciones y conexiones en uso)
  cuando en el último minuto alguna petición esperó por una conexión libre. Es la
  medición previa a subir `DATABASE_MAX_CONNS`: sin esperas, más conexiones no
  aceleran nada.
- El barrido de delivery no corre dos veces a la vez (ticker y cada `Claim`).

## Avisos en vivo (`pg_notify`)

La migración 000138 agrega `notify_live_change()` como trigger `AFTER … FOR EACH
ROW` en pedidos, platos y rondas de cocina, cobros, devoluciones, caja, avisos del
personal, pagos detectados de Fudia Pay, disponibilidad, saldos, productos,
despachos, mesas y reservas. Emite `pg_notify('foods_live', 'empresa:local:tema')`;
PostgreSQL lo entrega al confirmar y descarta los repetidos de la misma
transacción, así que el costo es una llamada por fila escrita y un aviso por tema y
transacción. Los platos toman el local de su pedido; los productos avisan a toda la
empresa (local vacío). `TestLiveChangesReachOnlyTheirLocationAfterCommit` comprueba
que el aviso llega solo tras el commit, una vez y solo al local correcto.

## Cobrado neto del pedido (`orders.paid_amount`)

Es un derivado de los documentos, nunca un dato que se escriba a mano. Los
triggers `payments_paid_amount` y `payment_refunds_paid_amount` recalculan por
completo (no por incrementos) la suma de cobros menos devoluciones del pedido en la
misma transacción que inserta, corrige o borra un cobro o una devolución, con el
mismo alcance de empresa y local que usaba el cálculo en línea. Los caminos de
cobro (`LockPaymentOrder`, `LockBatchOrder`) y de devolución (`LockRefundPayment`)
ya bloquean el pedido con `FOR UPDATE` antes de insertar, así que el trigger no
agrega bloqueos nuevos ni cambia el orden de bloqueo. La columna es `numeric` sin
escala para conservar el texto anterior de la API («0» sin cobros, «60.00» con
ellos). La migración carga los pedidos con cobros de una vez dentro de su
transacción (≈30 s con 460 mil pedidos en local); en producción con más volumen,
programarla fuera de la hora punta. `TestOrderPaidAmountFollowsPaymentsAndRefunds`
recorre cobro, devolución, corrección y borrado, compara la columna con los
documentos y verifica cada filtro de POS. Desde 2026-11-20 el dashboard también la usa
para decidir qué pedidos están pagados; los demás cálculos de cobrado (detalle de
pedido, Salón, montos del dashboard, cierre automático) siguen leyendo los documentos.

Las migraciones corren dentro de una transacción, por lo que `CREATE INDEX` bloquea
escrituras en la tabla mientras se construye; en tablas grandes de producción, crear
el índice con `CONCURRENTLY` fuera del runner antes de desplegar la migración, que
usa `IF NOT EXISTS`.

## Extensiones

Las migraciones crean `pgcrypto` y `pg_trgm`, pero el código de ejecución no
debe depender de funciones de `pgcrypto` sin calificar: tras restaurar un volcado,
la extensión puede quedar en un esquema fuera del `search_path` y la función deja
de resolverse solo en el entorno restaurado. Para valores aleatorios usar
`gen_random_uuid()`, nativo desde PostgreSQL 13. La prueba de integración
`TestTableWritesWorkWhenPgcryptoIsOutsideSearchPath` reproduce ese escenario
dentro de una transacción revertida. Los enlaces de delivery usan el mismo
mecanismo: `TestDeliveryLinksWorkWhenPgcryptoIsOutsideSearchPath` comprueba la
migración 115, la creación y regeneración del QR y la corrección 119 para bases
que ya tenían el valor predeterminado anterior, sin cambiar tokens existentes.
