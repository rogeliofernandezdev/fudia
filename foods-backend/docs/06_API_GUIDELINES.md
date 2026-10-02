# API

REST JSON descrito en api/openapi.yaml. Los errores usan código estable, mensaje
seguro y correlationId. Paginación, filtros, idempotencia y concurrencia se
definen en OpenAPI antes del frontend.

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
