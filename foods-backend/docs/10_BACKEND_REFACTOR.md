# Separación de responsabilidades — etapa 1

## Alcance y estado real

Base: `feat/ui-standard`, commit `f7ad296fca6a90d8e3886683b2a612e945f7bf90`.
El backend mezclaba HTTP, reglas, SQL y coordinación transaccional dentro de
`internal/platform/httpapi`. Esta entrega inicia una migración incremental;
no declara completada la arquitectura hexagonal de todo el backend.

| Carpeta | Responsabilidad | Dependencias permitidas |
| --- | --- | --- |
| `cmd/api` | Configuración y composición de adaptadores | Módulos y plataforma |
| `internal/modules/<módulo>/domain` | Modelos, normalización e invariantes | Biblioteca estándar, sin HTTP ni persistencia |
| `internal/modules/<módulo>/application` | Casos de uso e interfaces Repository | Dominio y contratos transversales |
| `internal/modules/<módulo>/infrastructure/postgres` | SQL, mapeo de filas y transacciones del agregado | Contratos de aplicación, dominio y pgx |
| `internal/modules/<módulo>/transport/http` | Decodificación, cookies y traducción de respuestas | Aplicación, dominio y primitivas HTTP |
| `internal/platform` | Configuración, conexiones, contexto autenticado y respuestas comunes | Sin importaciones de módulos ni legado |
| `internal/legacy/httpapi` | Implementación anterior pendiente de extracción | Excepción temporal explícita |
| `internal/architecture` | Pruebas que impiden dependencias prohibidas | Análisis estático de los imports y SQL |

El flujo es HTTP → caso de uso → interfaz Repository → implementación PostgreSQL.
La dependencia de compilación se invierte: PostgreSQL implementa la interfaz
que declara el caso de uso. No existe un repositorio CRUD genérico ni una capa
intermedia que solo renombra llamadas a pgx.

## Verticales extraídas

- **Identity / sesiones:** login, logout, autenticación y autorización por permiso.
  `SessionRepository` resuelve candidatos, acceso a locales y sesiones; el adaptador
  de credenciales encapsula bcrypt y generación de tokens. La cookie pertenece a
  HTTP; el caso de uso nunca recibe `http.Request` ni `http.ResponseWriter`.
  Usuarios, roles, perfil y cambio de contexto todavía están en el legado.
- **Customers:** repositorio del agregado cliente/direcciones, casos de uso de
  listado, consulta, alta, edición y estado. La sustitución de direcciones y la
  modificación del cliente se confirman juntas o se revierten juntas.
- **Inventory / unidades:** normalización de código/nombre, listado y alta.
  La creación de unidad, combinaciones de presentación y auditoría conserva una
  única transacción. Stock, entradas y compras siguen pendientes.
- **Delivery:** validación pura de contacto en `delivery/domain`; el flujo de
  pedidos sigue usando esa misma regla mediante un puente temporal.

Los handlers antiguos delegan estas operaciones a dependencias inyectadas. No
construyen repositorios. Producción los compone en `cmd/api/modules.go`; la
suite de integración tiene su propia composición de pruebas equivalente.

## Compatibilidad

Se conservan rutas, permisos, envelopes JSON, alcance de sesión, hash de tokens,
expiración, códigos de error y UUID generados por PostgreSQL. No hay cambios de
OpenAPI, migraciones, dependencias Go, frontend ni infraestructura de despliegue.
La opción `FOODS_COOKIE_SECURE` ahora se lee al componer el servidor.

La lectura de clientes propaga errores de iteración y de carga de direcciones;
ya no responde éxito con información parcial ante un fallo de PostgreSQL.
La auditoría de clientes conserva la política previa de mejor esfuerzo después
del commit; no se presenta como auditoría transaccional. Esa mejora requiere
una decisión y pruebas propias. Las unidades sí mantienen auditoría atómica.

## Validación

- `go test ./...`: casos de uso con repositorios en memoria, invariantes,
  respuestas y regresiones anteriores; las pruebas de PostgreSQL requieren
  `DATABASE_URL` y se omiten explícitamente si no está configurada.
- `go vet ./...` y `go build ./cmd/api`.
- `go test ./internal/architecture`: prohíbe HTTP/pgx/SQL en aplicación y dominio,
  persistencia en transporte e importaciones de módulos desde plataforma.
- PostgreSQL temporal en el workflow `foods-backend`: migraciones y suite completa,
  incluyendo clientes entre empresas, permisos reales y rollback de direcciones;
  las pruebas existentes cubren sesiones, unidades, inventario, pedidos y caja.

Los archivos de regresión se trasladan al legado sin cambiar la profundidad
relativa de sus accesos a `migrations` y `api/openapi.yaml`.

## Siguiente extracción

1. Completar Identity: usuarios, roles, perfiles y cambio de contexto.
2. Extraer catálogos de Menu y configuración de Organizations, incluyendo sus
   reglas de suscripción y puertos de colaboración explícitos.
3. Extraer Inventory/Purchasing, manteniendo una unidad transaccional que incluya
   documento, saldo, movimiento y auditoría.
4. Extraer Orders/Service/Kitchen/Billing coordinadamente. Sus repositorios deben
   compartir la transacción de cada caso de uso; no abrir una transacción nueva
   por repositorio ni separar pago, consumo y cierre en commits independientes.
5. Extraer Reporting, suscripciones e integración Concierge detrás de puertos.
6. Retirar puentes, composición y paquete `legacy` solo cuando sus verticales
   tengan pruebas y rutas equivalentes en los módulos nuevos.

Cada extracción debe reducir el legado y pasar los controles. Añadir carpetas
vacías, mover handlers mezclados a carpetas de módulo o crear interfaces que
exponen pgx al caso de uso no se considera separación terminada.
