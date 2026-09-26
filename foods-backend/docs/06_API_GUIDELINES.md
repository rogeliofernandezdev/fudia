# API

REST JSON descrito en api/openapi.yaml. Los errores usan código estable, mensaje
seguro y correlationId. Paginación, filtros, idempotencia y concurrencia se
definen en OpenAPI antes del frontend.

## Lookups de catálogo

Los endpoints usados por autocompletes exponen `q`, `page` y `pageSize` y devuelven `items`, `total`, `page` y `pageSize`. El backend nunca obliga al frontend a descargar el catálogo completo: la apertura puede pedir una página pequeña y las búsquedas posteriores recorren la paginación según necesidad.
