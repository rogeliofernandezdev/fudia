# Separación de responsabilidades y patrón Repository

## Estado y alcance

Migración completada en `feat/ui-standard`. Se retiró `internal/legacy`.
Los flujos existentes del backend utilizan casos de uso y repositorios específicos.
El enrutador ya no tiene un pool, consultas SQL ni lógica de persistencia.

| Carpeta | Responsabilidad | Dependencias |
| --- | --- | --- |
| `cmd/api` | Configuración y composición de producción | Módulos, adaptadores y plataforma |
| `internal/transport/httpapi` | Rutas, autorización y delegación a transports de módulos | Contratos HTTP y contexto autenticado |
| `internal/modules/<módulo>/domain` | Modelos, normalización e invariantes puras | Biblioteca estándar; sin HTTP, SQL ni otros módulos |
| `internal/modules/<módulo>/application` | Casos de uso, interfaces Repository y coordinación transaccional | Dominio, puertos colaboradores y contratos compartidos |
| `internal/modules/<módulo>/infrastructure/postgres` | Consultas, bloqueos y mapeo de filas a resultados tipados | Contratos de aplicación, dominio y pgx |
| `internal/modules/<módulo>/transport/http` | Parámetros, JSON, cookies, archivos y traducción de errores | Aplicación, dominio y primitivas HTTP |
| `internal/platform` | Configuración, conexiones, UnitOfWork, contexto, errores y mecánica HTTP | Sin módulos de negocio |
| `internal/architecture` | Control automático de imports, SQL fuera de adaptadores y ausencia de legacy | Análisis estático |

El flujo es HTTP → caso de uso → interfaz Repository → adaptador PostgreSQL.
Los repositorios devuelven entidades, proyecciones o resultados tipados; no exponen
`pgx.Row`, `pgx.Tx` ni operaciones SQL genéricas a aplicación. Los adaptadores de
archivos, credenciales y diagnóstico también se inyectan desde composición.

## Responsabilidad por módulo

| Módulo | Flujos |
| --- | --- |
| `identity` | Sesiones, autenticación, permisos, usuarios, roles, perfiles, contexto y protección del último administrador |
| `customers` | Clientes y agregado de direcciones |
| `menu` | Categorías, productos, imágenes, alérgenos, combos, modificadores, recetas, disponibilidad e historial; proyecciones públicas para Concierge |
| `inventory` | Unidades, presentaciones, entradas, ajustes, saldos, movimientos valorados, transferencias y consumo/reversión por ventas y recetas |
| `purchasing` | Proveedores, órdenes de compra, aprobación, recepciones parciales, devoluciones e idempotencia de documentos |
| `orders` | Creación y edición, cálculo de precios, selecciones, rondas, cocina/barra, entrega, cuentas de salón y pedidos idempotentes del QR |
| `service` | Zonas, mesas, QR, lotes y reservas |
| `billing` | Medios de pago, gastos, cobros individuales/divididos, devoluciones, cajas, turnos, asignaciones, operaciones, arqueos e informes |
| `organizations` | Datos de empresa, perfiles fiscales, locales, tipos de cambio, catálogos de país/moneda y puesta en marcha |
| `subscriptions` | Planes, contratos, condiciones, periodos, cobros SaaS, límites y activación de módulos |
| `platformadmin` | Administración global, canales de WhatsApp, consultas de empresas y alta atómica con sus plantillas iniciales |
| `concierge` | Autenticación del servicio en HTTP, resolución de QR, coordinación con Menu/Orders, solicitud de cuenta y avisos al personal |
| `reporting` | Dashboard y proyecciones operativas con snapshot coherente |
| `audit` | Puerto y adaptador de auditoría compartida |
| `delivery` | Validación pura de contacto utilizada por Orders |

Cocina pertenece al cumplimiento de pedidos en `orders`. La configuración de
recetas pertenece a `menu`; su consumo y valorización pertenecen a `inventory`.
Concierge no mantiene una segunda implementación del precio ni del descuento de stock.
El alta de una empresa coordina sus datos, plantillas, administrador y contrato;
las reglas de planes y módulos permanecen en `subscriptions`.

## Transacciones y colaboración

`platform/unitofwork` declara `Manager` y `Transaction` sin importar pgx.
El adaptador PostgreSQL vincula un contexto a una transacción y al pool que la creó.
Todos los repositorios del mismo pool que reciben ese contexto usan esa transacción.
Se rechazan transacciones anidadas.

- Pedidos, rondas, selecciones y consumo de stock/recetas se confirman o revierten juntos.
- Recepciones, devoluciones y transferencias mantienen documento, saldo y movimientos
  dentro de la misma transacción, incluyendo las comprobaciones de idempotencia.
- Cobros y devoluciones conservan las comprobaciones de saldo y los movimientos de caja;
  el cierre automático del pedido participa en la transacción del cobro.
- Cerrar caja guarda conteo, informe, estado, liberación de usuarios y auditoría juntos.
- El alta de empresa incluye perfiles, local, administrador, plantillas, contrato,
  asignación y auditoría antes del commit.
- Reporting e informes de caja conservan snapshots de solo lectura/repeatable-read.

La política previa de auditoría de mejor esfuerzo se conserva donde ya existía.
No se presenta como transaccional una auditoría que ocurre después del commit.
`database.WithTransaction` permite vincular una transacción externa desde infraestructura
o fixtures de integración; los casos de uso emplean exclusivamente `unitofwork.Manager`.

## Compatibilidad

Se conservan las rutas, permisos, campos JSON, alcance por empresa/local, generación
de UUID persistidos en PostgreSQL, cookies y contratos operativos. No se modificaron
OpenAPI, migraciones, dependencias Go, frontend ni el workflow de despliegue.

Los archivos de regresión residen en `internal/transport/httpapi`, con la misma
profundidad relativa para fixtures de `migrations` y `api/openapi.yaml`. Los puentes
`*_bridge_test.go` conservan nombres usados por esas pruebas y delegan en las
implementaciones nuevas. La composición de pruebas mantiene un pool en un wrapper
exclusivo de tests; la API de producción recibe solamente handlers.

## Verificación

- `go test ./...`: reglas, casos de uso, contratos y regresiones existentes.
- `go vet ./...` y `go build ./...`.
- `go test ./internal/architecture`: dominio sin HTTP/persistencia, aplicación sin
  adaptadores, transport sin SQL/pgx y prohibición de reintroducir legacy.
- Workflow `foods-backend`: PostgreSQL 16 temporal, todas las migraciones, suite de
  integración, vet y build. Sin `DATABASE_URL`, las pruebas de integración locales
  se omiten explícitamente; no equivalen a la verificación con PostgreSQL.

La suite comprueba aislamiento entre empresas/locales, precios autoritativos,
idempotencia, inventario y recetas, recepciones/devoluciones, permisos, flujo de
salón, cierre/cobro concurrentes, caja y alta/suscripciones/Concierge.
