# Foods Backend

API oficial en Go. Antes de implementar, leer AGENTS.md y todos los documentos de docs/.

Verificación: go test ./..., go vet ./... y go build ./cmd/api.

Imágenes GCP: carga directa firmada, sin bytes en el API; el proyecto independiente
`../fudia-image-processor` genera WebP de 200/600 px. Contrato y configuración en
[docs/11_PRODUCT_IMAGES.md](docs/11_PRODUCT_IMAGES.md).

## Base de datos local

El proceso exige `DATABASE_URL` con `sslmode=require`. Copiar `.env.example` a un
archivo local ignorado por Git o inyectar el secreto desde el entorno. Aplicar
`migrations/000001_foundation.up.sql` con una herramienta PostgreSQL autorizada
antes de iniciar el API. El frontend nunca se conecta directamente a PostgreSQL.

Desde `foods-backend`, ejecutar `go run ./cmd/migrate`. El comando lee `.env`,
registra versiones en `schema_migrations` y aplica cada migración una sola vez
dentro de una transacción.

Rutas iniciales: `/health`, `/v1/auth/login`, `/v1/admin/dashboard` y
`/v1/admin/products`. El alcance de organización y local se deriva de la sesión.


## Fudia Concierge

La plataforma mantiene por país únicamente el número público, nombre visible y
estado del canal. `WHATSAPP_PHONE_ID`, los tokens, el secreto de la aplicación y
el token de verificación de Meta / WhatsApp Business se inyectan en Fudia
Concierge mediante variables de entorno; nunca se guardan ni se exponen en la
configuración administrativa. Cada país define también su moneda predeterminada
y el onboarding exige esa asociación.

`FUDIA_CONCIERGE_API_KEY` autentica las llamadas internas desde el servicio Concierge. El acceso de una organización se controla mediante `organization_modules.whatsapp_bot`, administrado únicamente desde la plataforma global.

## Responsabilidades y Repository

La implementación usa repositorios específicos en `internal/modules`. Cada caso de
uso depende de contratos tipados; los adaptadores PostgreSQL contienen las consultas
y el mapeo de filas. `cmd/api/modules.go` conecta todas las dependencias de producción.

`internal/transport/httpapi` registra las rutas y aplica autorización, sin SQL ni
conexiones a la base de datos. `internal/platform` contiene mecanismos compartidos.
El paquete anterior `internal/legacy` fue retirado. Las responsabilidades, la unidad
transaccional y las verificaciones están en
[docs/10_BACKEND_REFACTOR.md](docs/10_BACKEND_REFACTOR.md).
