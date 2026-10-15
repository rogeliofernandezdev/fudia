# Base de datos

PostgreSQL es la fuente de verdad. Las migraciones son versionadas y se aplican
hacia adelante. Las claves foráneas son UUID sin autogeneración.

Pedido, stock, pago y emisión usan transacciones explícitas. No se modifica stock
o saldo sin documento y movimiento asociado.

## Performance

Los endpoints paginados deben evitar N+1 y consultas por fila. Antes de agregar
caché, se reduce el número de round trips y se agrupan cálculos en SQL cuando sea
seguro.

Las búsquedas `ILIKE '%texto%'` de catálogos operativos usan `pg_trgm` e índices
GIN sobre los campos realmente consultados. Los filtros por tenant/local/estado
usan índices B-tree o parciales acordes al patrón de lectura.

`pageSize` permanece acotado. La paginación por offset es válida para los
listados administrativos actuales; cuando una tabla crezca hasta volver costosos
los offsets altos, se migra ese endpoint a keyset/cursor conservando un orden
estable.

Las consultas identificadas como lentas deben revisarse con `EXPLAIN (ANALYZE,
BUFFERS)` sobre un volumen representativo antes de introducir índices adicionales.

## Extensiones

Las migraciones crean `pgcrypto` y `pg_trgm`, pero el código de ejecución no
debe depender de funciones de `pgcrypto` sin calificar: tras restaurar un volcado,
la extensión puede quedar en un esquema fuera del `search_path` y la función deja
de resolverse solo en el entorno restaurado. Para valores aleatorios usar
`gen_random_uuid()`, nativo desde PostgreSQL 13. La prueba de integración
`TestTableWritesWorkWhenPgcryptoIsOutsideSearchPath` reproduce ese escenario
dentro de una transacción revertida.
