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
dentro de una transacción revertida.
