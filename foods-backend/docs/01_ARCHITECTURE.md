# Arquitectura

Monolito modular en Go con arquitectura hexagonal y vertical slices.
El dominio no importa HTTP, PostgreSQL ni proveedores externos. Los módulos se
componen en cmd/api. OpenAPI es la frontera oficial con los clientes.

Todos los flujos del backend usan los módulos y repositorios descritos en
`10_BACKEND_REFACTOR.md`. `platform` contiene mecanismos compartidos y
`internal/transport/httpapi` compone las rutas y sus permisos. El paquete
`internal/legacy` fue retirado.

`platform/realtime` reparte los avisos de cambio a las pantallas abiertas. La
migración 000138 agrega triggers que emiten `pg_notify('foods_live',
'empresa:local:tema')` al escribir en las tablas operativas; `cmd/api` mantiene
una conexión dedicada con `LISTEN` (sacada del pool) y el endpoint
`GET /v1/admin/events` transmite por SSE solo los temas del local de la sesión,
agrupados cada 250 ms, con un ping cada 25 s y cierre a los 15 minutos (el
navegador reconecta). El aviso no lleva datos: cada pantalla vuelve a leer por su
endpoint con sus permisos. Si se pierde la escucha, el endpoint responde 503 y las
pantallas siguen con su consulta periódica. Requiere conexión directa a
PostgreSQL: un PgBouncer en modo transacción acepta `LISTEN` pero no entrega los
avisos (las pantallas lo cubren con su consulta de respaldo de 60 s). La ruta queda
fuera del plazo de 30 s por petición y del aviso de petición lenta.
