# WhatsApp por país y transporte multiproveedor

Implementación de repositorio del 2026-11-08 sobre `feat/ui-standard`. No se ha
hecho commit, push, deploy, vinculación de dispositivo ni envío a clientes reales.
Se conservaron los cambios locales existentes. Esta guía describe lo implementado
y distingue las pruebas locales de la aceptación externa pendiente.

## Funcionalidades

- Meta Cloud API sigue siendo el proveedor predeterminado y conserva su adaptador
  y webhook HMAC. whatsmeow es una alternativa elegida manualmente por Fudia.
- Un único registro actual por país en `platform_whatsapp_channels`; sin números
  secundarios, prioridades ni failover. Los duplicados se archivan sin borrado.
- El número se verifica contra un activo autorizado de Meta o una sesión realmente
  vinculada antes de activarlo. Revisión optimista evita sobrescribir otra edición.
- La pantalla existente Países y WhatsApp permite proveedor, vinculación privada,
  número real, estado, activación y auditoría. Cambiar número/proveedor requiere
  confirmar una instantánea concreta del cambio. Solo el administrador global.
- `/api/public/concierge/{qr}` permanece: cada scan resuelve país/canal actual,
  genera wa.me dinámico y conserva los códigos temporales existentes. No se
  regeneran tokens de mesas, mostradores o delivery por cambiar el WhatsApp.
- Concierge conserva Python, Gemini, Redis Streams, secuencias, leases, fencing,
  efectos idempotentes, recovery y outbox. Los IDs internos incluyen proveedor y
  activo/sesión; la compatibilidad con IDs Meta anteriores no crea colisiones nuevas.
- El gateway Go solo transporta. Vinculación por QR, SQL store de whatsmeow,
  recuperación tras reinicio, reconexión, exclusión entre procesos, recepción de
  texto/ubicación, envío de texto/imágenes y recibos. JPEG/PNG y conversión de WebP
  existente, sin mover la IA al gateway.
- Cada envío nuevo del gateway espera aleatoriamente **2,5 a 5 segundos**, una vez
  preparada la respuesta con Gemini. El temporizador es cancelable. Las consultas
  idempotentes no vuelven a esperar ni vuelven a enviar un mensaje registrado.

## Archivos principales

| Área | Archivos nuevos o ampliados |
| --- | --- |
| Gateway independiente | `fudia-whatsapp-gateway/{go.mod,go.sum,Dockerfile,.env.example,AGENTS.md,README.md,api/openapi.yaml}`, `cmd/gateway/main.go`, `internal/gateway/{config,gateway,http,store,events,send,delay}.go`, `schema.sql`, pruebas y fixture WebP licenciado |
| Configuración Go | `foods-backend/internal/modules/platformadmin/{domain,application,infrastructure/postgres,transport/http}/whatsapp.go`, `contracts/whatsapp.go`, `providers/concierge/client.go`, tipos y métodos existentes de Plataforma |
| Visitas y directorio | `foods-backend/internal/modules/concierge/{domain,application,infrastructure/postgres,transport/http}/visits.go`, apertura de conversaciones y resolución de QR existentes |
| Rutas y composición | `foods-backend/internal/transport/httpapi/{routes,handlers}.go`, `cmd/api/modules.go`, contrato `api/openapi.yaml` |
| Canal de avisos | Tipos y consultas de `notifications` y `delivery`; triggers SQL; `fudia-concierge/src/services/{customer_notices,delivery_partners,outbox,queue_worker}.py` |
| Concierge | `src/{domain,api,infrastructure}/whatsapp*.py`, `src/services/restaurants.py`, `application.py`, `config/settings.py`, webhook/start, cliente Fudia, cola, modelos y herramientas existentes |
| Administración | `foods-admin-web/src/modules/platform/presentation/{platform-global-settings-page,whatsapp-channel-editor}.tsx`, CSS del formulario, tipos/API, BFF con no-store |
| Infraestructura | `foods-infrastructure/whatsapp-gateway/compose.yaml`, `gcp/concierge-deploy.mjs`, referencias opcionales en workflow Concierge, `docs/13_WHATSAPP_GATEWAY.md` |
| Validación | Go: `whatsapp_multiprovider_integration_test.go` y `whatsapp_migration_integration_test.go`; Python: `test_whatsapp_multiprovider.py` e integración Redis; gateway: `gateway_test.go`; preview local del editor y contratos existentes |

No se enumeran como parte de este cambio las modificaciones locales de otros
módulos que ya existían en el workspace.

## Migraciones

| Migración | Resultado |
| --- | --- |
| `000120_whatsapp_providers` | Proveedor Meta por defecto, una configuración actual por país, revisiones, verificación, archivo de duplicados, auditoría y referencias inmutables de transporte |
| `000121_concierge_visits_and_origins` | Identidades verificadas/alias, visitas con UUID reales de empresa/local y origen inmutable de conversación |
| `000122_whatsapp_notice_routing` | Canal/destinatario fijados al crear avisos de cliente y canal fijado en mensajes a centrales |

Se validaron en PostgreSQL 17 temporal, además del ensayo de migración de duplicados
sobre el esquema anterior a 120. No se ejecutaron sobre la base del negocio.
La migración 067 ya retiró phone_number_id: no se intenta recuperarlo de una columna
inexistente. Concierge importa el activo Meta original desde su configuración
privada y lo verifica. Solo restaura automáticamente un único activo anterior cuyo
número coincida. Varios activos previos requieren revisión manual. Si falta la
verificación, queda inactivo y pendiente; nunca se activa un proveedor alternativo.

Los down de 120/121 conservan auditoría e identidades; no borran ese historial para
revertir una aplicación. El down de 122 retira triggers, conserva referencias.
Revisar compatibilidad de la versión anterior antes de cualquier rollback.

## Endpoints

Contrato público de componentes: `foods-backend/api/openapi.yaml`.

| Método/ruta | Acceso |
| --- | --- |
| GET/POST `/v1/platform/whatsapp-channels` | Administrador global; listado con catálogo de proveedores y alta única por país |
| PATCH `/v1/platform/whatsapp-channels/{id}` | Administrador global; edición del mismo registro, verificación y revisión |
| POST `/v1/platform/whatsapp-channels/link` | Administrador global; inicio/consulta privada del QR de dispositivo |
| GET `/v1/platform/whatsapp-channels/{id}/history` | Administrador global; historial sin credenciales |
| GET `/v1/integrations/concierge/whatsapp/routes` | Clave de integración; referencias actuales/históricas |
| POST `/v1/integrations/concierge/whatsapp/legacy-route` | Clave de integración; importación inmutable del activo anterior |
| POST `/v1/integrations/concierge/restaurants/search` | Clave de integración; visitas o descubrimiento elegible del país |
| POST `/v1/integrations/concierge/restaurants/context` | Clave de integración; revalidación del local y QR sin rotarlo |

La apertura existente de conversación acepta canal, destinatario e identidad y
registra origen/visita en la misma transacción. El QR público agrega país y UUID
reales. Los avisos devuelven `channelKey` y destinatario original.

Concierge recibe `/internal/whatsapp/{verify,link}` con la clave privada del backend,
y `/internal/whatsapp/gateway/events` con otra clave exclusiva de ingress.
El gateway expone privadamente `/v1/link`, `/v1/sessions/{id}`, `/v1/messages` y
`/metrics`; contrato en su `api/openapi.yaml`. `/live` y `/ready` son mínimos.

## Configuración y uso

Meta conserva `WHATSAPP_TOKEN`, `WHATSAPP_PHONE_ID`, `WHATSAPP_VERIFY_TOKEN`,
`WHATSAPP_APP_SECRET` y `WHATSAPP_GRAPH_VERSION`. `WHATSAPP_COUNTRY_CODE` asocia el
activo heredado a su país (PE por defecto). Opcionalmente, `WHATSAPP_META_ACCOUNTS`
es un **secreto** JSON con objetos `{countryCode,phoneId,token}` para activos
autorizados/retenidos. No registrar dos veces el activo heredado. Esta allowlist
no crea canales operativos y no se envía al frontend. Verificar número, registro
y disponibilidad del activo real; editar PostgreSQL no registra un número en Meta.

Gateway: base SQL exclusiva protegida, dos claves de servicio de al menos 32
caracteres, URL de Concierge y hosts de imágenes autorizados. En Concierge,
`WHATSAPP_GATEWAY_URL`, `WHATSAPP_GATEWAY_KEY` y `WHATSAPP_GATEWAY_INGRESS_KEY`.
En backend, `FUDIA_CONCIERGE_URL` y su clave de integración existente. Mantener
URLs y bases del mismo entorno. Referencias de despliegue y protección en
`foods-infrastructure/docs/13_WHATSAPP_GATEWAY.md`.

En Países y WhatsApp se edita el registro existente. Al elegir whatsmeow se pulsa
Vincular WhatsApp, se escanea el QR desde Dispositivos vinculados, se verifica el
número real y se guarda/activa. El QR privado no es el QR de mesa. Nunca se muestran
claves, archivos de sesión ni identificadores técnicos del proveedor al usuario.

## Selector e identidad

El historial PostgreSQL guarda empresa/local, última visita, cantidad y vínculo a
un cliente existente cuando su PN está verificado. No se crean clientes duplicados
para hacer seguimiento. La memoria Redis anterior se revalida por su QR e IDs reales
y abre una primera visita persistente; no se pierde el último restaurante conocido
durante la actualización. Un LID sigue siendo opaco aunque solo contenga números.
Un PN autenticado puede vincular un alias existente, nunca reasignarlo a otra persona.
Un teléfono escrito para contacto del pedido no prueba identidad ni permite leer
un perfil o modificar preferencias de terceros.

Antes de invocar Gemini: una mesa vigente continúa; después de cierre o tres horas
de inactividad, un solo restaurante visitado elegible abre delivery/recojo. Varios
muestran lista numerada, paginación y `AWAITING_RESTAURANT_SELECTION` en Redis con
UUID reales. Sin visitas se ofrece descubrimiento del país, nunca una empresa
arbitraria. Se revalida la opción antes de entrar y cargar carta/disponibilidad.

Cambiar de restaurante crea otro contexto y limpia datos temporales. No cancela
pedidos ni mezcla empresas. Una mesa vigente, un handoff o una operación comercial
incierta impiden cambiar. Un carrito con confirmación aún no enviada se puede
descartar explícitamente. Se conservan las barreras por secuencia y fecha de envío.

## Cambio de número y avisos

Los scans nuevos resuelven la configuración actual; los enlaces wa.me compartidos
directamente antes no pueden cambiar. Las conversaciones/pedidos antiguos mantienen
sus referencias y no se trasladan al número nuevo. La entrada por un activo inactivo
no inicia atenciones nuevas; el cliente debe escanear el QR vigente.

Respuestas, imágenes, avisos y ofertas pendientes resuelven su canal original,
no el actual del país. Si falta ese transporte se conserva el error/pendiente;
no se envía desde otro número. Mantener credenciales/sesiones históricas mientras
existan mensajes pendientes. Meta conserva su ventana de mensajería; las tres horas
de mesa son una regla independiente. whatsmeow no reutiliza plantillas de Meta;
las opciones a centrales usan texto numerado citando el mensaje original.

## Validación y límites

- Backend: suite Go completa con PostgreSQL temporal, arquitectura, vet y build.
- Gateway: 8 pruebas, vet y build; espera real superior a dos segundos, aleatoriedad,
  autenticación, JID/LID, ubicación, propietario exclusivo, SQL/reinicio, texto,
  imagen/WebP, idempotencia y envío incierto con transporte simulado.
- Concierge: ruff sin errores, mypy (61 archivos), 306 pruebas aprobadas con Redis
  real temporal, una prueba de carga omitida y build del paquete.
- Administración: lint sin errores (7 advertencias previas), typecheck,
  727 pruebas aprobadas y build. Navegador real con
  fixtures de transporte a 390×844, 390×480 y 1440×900: sin overflow horizontal,
  QR completo mediante scroll interno, pie accesible, controles activos de al menos
  44 px e interruptor de 52 px. Bloqueo visible de activación antes de vincular.
- Infraestructura: 64 pruebas de configuración/simulaciones; imagen Docker local
  construida y arrancada, `/ready` correcto, healthcheck healthy y usuario sin root.
- Los contratos OpenAPI se parsearon y se validaron sus referencias de schemas.

La aceptación **externa** sigue pendiente: alta real Meta, vinculación con un
WhatsApp autorizado, reinicio y reconexión real, mensajes/ubicaciones/recibos,
Gemini con credenciales y pedido/KDS/cuenta extremo a extremo en staging.
Los dobles y la recuperación SQL no certifican una sesión real de WhatsApp.
whatsmeow es no oficial y puede perder compatibilidad; no se usa para eludir
restricciones. No se desplegaron VM, SQL, IAM ni secretos; la protección administrada
del almacenamiento debe verificarse por el operador. No se han medido capacidad,
SLA ni un teléfono físico. La prueba de carga Redis se mantiene optativa.
