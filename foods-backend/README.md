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

El canal de WhatsApp es global para toda la plataforma. El número oficial se administra exclusivamente en Meta / WhatsApp Business y no se duplica en la configuración de foods-backend.

`FUDIA_CONCIERGE_API_KEY` autentica las llamadas internas desde el servicio Concierge. El acceso de una organización se controla mediante `organization_modules.whatsapp_bot`, administrado únicamente desde la plataforma global.
