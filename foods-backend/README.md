# Foods Backend

API oficial en Go. Antes de implementar, leer AGENTS.md y todos los documentos de docs/.

Verificación: go test ./..., go vet ./... y go build ./cmd/api.

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
