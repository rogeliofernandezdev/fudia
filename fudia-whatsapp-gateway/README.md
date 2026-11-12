# Fudia WhatsApp Gateway

Transporte independiente con `go.mau.fi/whatsmeow`, versión exacta en `go.mod`.
Concierge conserva Python, Gemini, pedidos, herramientas y Redis Streams. Meta
sigue siendo el proveedor predeterminado. whatsmeow es una biblioteca no oficial:
su compatibilidad puede cambiar; no utilizarla para eludir restricciones de WhatsApp.

## Configuración

Crear una base PostgreSQL exclusiva para sesiones, con usuario exclusivo, backups
cifrados y acceso restringido. El esquema propio y el de whatsmeow se actualizan
al arrancar. Los UUID de Fudia los genera PostgreSQL. Las claves de dispositivo
se guardan en el SQL store de la biblioteca; no se exportan a archivos ni al navegador.
La protección en reposo corresponde al almacenamiento administrado cifrado y sus
backups, no a un cifrado adicional por fila implementado por este servicio.

En producción se exigen `SESSION_STORAGE_PROTECTED=true` y `DATABASE_URL` con
`sslmode=verify-full` o `verify-ca`, CA del servidor y permisos mínimos. Esta
declaración requiere comprobar el cifrado y acceso del proveedor de SQL. Local:
copiar `.env.example` a un archivo privado e inyectar sus variables al proceso;
el ejecutable no carga `.env` automáticamente.

| Variable | Uso |
| --- | --- |
| `DATABASE_URL` | Base exclusiva, sin compartir sesiones entre entornos |
| `GATEWAY_API_KEY` | Secreto de al menos 32 caracteres para Concierge → gateway |
| `CONCIERGE_INGRESS_KEY` | Otro secreto de al menos 32 caracteres para gateway → Concierge |
| `FUDIA_CONCIERGE_URL` | Origen HTTPS del ingress de Concierge |
| `MEDIA_ALLOWED_HOSTS` | Hosts exactos autorizados para imágenes de carta; separados por coma |
| `LISTEN_ADDRESS` | `:8020` por defecto |
| `APP_ENV` | Solo `development` permite HTTP/TLS SQL local para pruebas |

No imprimir conexiones, claves, QR privados, teléfonos ni mensajes. En producción
inyectar secretos desde Secret Manager con la identidad del servicio. La imagen
se ejecuta sin root y contiene una comprobación `/ready`.

## API privada y vinculación

Contrato en [api/openapi.yaml](api/openapi.yaml). Todas las operaciones y métricas
requieren `X-Fudia-Gateway-Key`; únicamente `/live` y `/ready` son públicos y mínimos.

- `POST /v1/link`: `{countryCode}` inicia el QR de dispositivo temporal en memoria.
- `GET /v1/sessions/{sessionRef}?countryCode=PE`: estado, número real y QR vigente.
- `POST /v1/messages`: sesión original, destinatario PN/JID/LID, texto o imagen,
  `callbackRef` SHA-256 estable. No admite grupos ni broadcasts.
- `GET /metrics`: contadores sin etiquetas de teléfonos, países ni contenido.

El administrador global usa **Plataforma → Países y WhatsApp → whatsmeow →
Vincular WhatsApp**. Escanea desde Dispositivos vinculados. Solo se muestra conectado
después del evento autenticado de conexión; el número real debe coincidir al activar.
El QR privado nunca se incluye en el QR público de mesa.

## Entrega y espera aleatoria

Antes de cada envío nuevo se espera aleatoriamente **2,5 a 5 segundos**, después de
que Concierge prepare la respuesta con Gemini. La espera usa un temporizador
cancelable, sin bloquear el procesamiento de otras conversaciones. También se
aplica a imágenes y avisos que usen este transporte; Meta conserva su comportamiento.
Las consultas idempotentes de un envío registrado no repiten la espera ni el mensaje.

El gateway persiste identificador externo y hash antes de enviar. Un envío incierto
queda `uncertain`: no se vuelve a enviar a ciegas. Un recibo reconcilia su referencia
original. Las respuestas de texto numeradas a ofertas deben citar el mensaje original.
No se confunden sus números de selección con los del selector de restaurantes.

Las imágenes WebP existentes se decodifican y convierten a JPEG/PNG antes de
enviar, con límite de bytes/píxeles y dos decodificaciones simultáneas.

La entrada validada se persiste en un outbox SQL y se entrega en orden al endpoint
privado `/internal/whatsapp/gateway/events` de Concierge. Un reintento usa el mismo
identificador. Después de aceptación se limpia el contenido del outbox. Los LID
son identificadores opacos: solo un PN autenticado o el mapa verificado de la
biblioteca acredita un teléfono. La aplicación no extrae teléfonos del texto del cliente.

## Operación y validación

Un único proceso posee la base mediante advisory lock SQL sobre conexión dedicada.
Un segundo proceso falla al arrancar. La pérdida del lock bloquea envíos y desconecta
sesiones. No usar un pooler en modo transaction para esta conexión. Reiniciar mantiene
credenciales SQL y recupera sesiones; logout exige nueva vinculación administrativa.
Las referencias históricas permiten terminar avisos antiguos y no son configuraciones
operativas adicionales por país.

Ejecutar `go test ./...`, `go vet ./...`, `go build ./...`. Con una base temporal,
`GATEWAY_TEST_DATABASE_URL` activa pruebas de propiedad exclusiva, recuperación SQL,
outbox, envíos de texto e imagen e idempotencia con transporte simulado. Estas pruebas
no vinculan un teléfono ni establecen una sesión real en WhatsApp. Debe probarse
vinculación, reinicio, ubicación, envío y recibos con un dispositivo autorizado en staging.

Infraestructura continua: [runbook](../foods-infrastructure/docs/13_WHATSAPP_GATEWAY.md).
API actual consultada: [whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow),
[SQL store](https://pkg.go.dev/go.mau.fi/whatsmeow/store/sqlstore).
