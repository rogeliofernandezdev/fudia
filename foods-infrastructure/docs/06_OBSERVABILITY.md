# Observabilidad

Logs JSON, métricas y trazas comparten correlación. Alertas cubren API, PostgreSQL,
impresión, comprobantes y retrasos de cocina. No registrar información sensible.

## Requests HTTP

Cada request de API emite `X-Request-ID`; el mismo identificador aparece como
`correlationId` en errores y como `request_id` en logs.

El logger registra `duration_ms`. Los requests que superan el umbral
`SLOW_REQUEST_MS` (500 ms por defecto) se registran como
`http_request_slow`. Este dato es la entrada para revisar SQL, payload, red o
procesamiento antes de optimizar.

Objetivo operativo inicial: endpoints comunes por debajo de 300 ms de backend y
listados normales por debajo de 500 ms bajo carga representativa. Estos valores
son objetivos de ingeniería, no sustituyen mediciones de producción.
