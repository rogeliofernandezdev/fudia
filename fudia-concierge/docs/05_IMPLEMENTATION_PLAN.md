# Plan de implementación

## Bloque 1 — Fundación

- [x] Proyecto en monorepo.
- [x] FastAPI, liveness y readiness.
- [x] Webhook WhatsApp con firma.
- [x] Adaptador WhatsApp.
- [x] OpenAI Responses API con tools (proveedor alternativo).
- [x] Gemini en Vertex AI como proveedor por defecto, con Flash-Lite (router, carta, servicio) y Flash (pedido) y escalado seguro.
- [x] Timeouts, reintentos y topes del modelo; instrucciones reenviadas en cada vuelta de tools; aviso al cliente en dead-letter.
- [x] LangGraph.
- [x] Redis para sesión/carrito.
- [x] Cliente Fudia.
- [x] Confirmación explícita protegida por código.
- [x] Tests y CI.

## Bloque 2 — Contrato Fudia

- [x] Menú server-to-server derivado del QR.
- [x] Creación de pedido derivada del QR.
- [x] Revalidación de precio, disponibilidad y stock en foods-backend.
- [x] Integración KDS con canal WhatsApp.
- [x] Idempotencia por ronda con `conversationId` estable y `requestId` independiente.
- [x] Credencial propia para el contrato Concierge -> foods-backend.
- [x] Test QR -> pedido -> KDS.
- [x] Cuenta acumulada desde backend.
- [x] Handoff humano persistente.

## Bloque 3 — QR y configuración

- [x] Un único bot global; el número oficial permanece administrado exclusivamente en Meta / WhatsApp Business.
- [x] Botón Pedir por WhatsApp en página QR.
- [x] Deeplink con token.
- [x] Entitlement `whatsapp_bot` activable únicamente por administrador global para cada empresa.
- [x] Contrato server-to-server bloqueado cuando la empresa no tiene `whatsapp_bot` activo.
- [x] Sesión conversacional rechaza QR de empresa sin Concierge habilitado.
- [x] Deeplink resuelto por Concierge consultando a Meta desde `WHATSAPP_PHONE_ID`, sin número duplicado en Fudia.
- [x] Admin Web con loading, error, validación y permisos.
- [x] Layout adaptable en la configuración y página pública de mesa.

## Bloque 4 — Conversación avanzada

- [x] Combos conversacionales.
- [x] Modificadores.
- [x] Nuevas rondas sobre una misma comanda aunque rondas anteriores estén preparando/listas.
- [x] KDS independiente por ronda.
- [x] Solicitud de cuenta con consumo, pagos y saldo reales.
- [x] Handoff humano resoluble desde Operaciones.
- [x] Bandeja operativa conectada a datos reales.
- [x] Módulo `whatsapp_bot` activable por entitlement.

## Bloque 5 — Endurecimiento de producción en código

- [x] Router unificado `out_of_scope/menu/order/service` con una sola llamada IA de clasificación por mensaje.
- [x] Tools separadas por dominio y mínimo privilegio aplicado en código.
- [x] Redis Streams en lugar de `BackgroundTasks`.
- [x] Concurrencia acotada por worker con backpressure al leer la cola.
- [x] Renovación de visibilidad para mensajes en vuelo.
- [x] Shutdown con drenaje de mensajes en vuelo.
- [x] Deduplicación atómica por mensaje.
- [x] ACK, reintentos y dead-letter.
- [x] Lock distribuido renovable por conversación.
- [x] Sesiones aisladas por `phone_number_id + cliente`.
- [x] Rate limiting y límite de tamaño de entrada.
- [x] Timeouts y retries seguros/idempotentes.
- [x] Fail-fast de configuración en producción.
- [x] `/live`, `/ready` y `/metrics`.
- [x] Correlation tracing.
- [x] Métricas Prometheus sin PII como labels.
- [x] Eval corpus de scope, intents y prompt injection.
- [x] Tests de separación de privilegios.
- [x] Dependencias fijadas + constraints reproducibles.
- [x] Contenedor no-root con healthcheck.
- [x] CI valida lint, tipos, tests, build Python y build Docker.

## Bloque 6 — Recuperación y consistencia operativa

- [x] Lock con fencing: la pérdida del lease cancela el trabajo y bloquea escrituras.
- [x] Registro de procesamiento por mensaje con puntos de recuperación.
- [x] Procesamiento y envío separados; un fallo de envío no repite tools.
- [x] `requestId` persistido antes de crear la ronda y reutilizado al recuperar.
- [x] Orden por conversación con secuencia asignada al encolar.
- [x] Cambio de QR: mensajes y respuestas del contexto anterior no se aplican ni se envían.
- [x] Envío incierto a WhatsApp reconciliado con estados de Meta.
- [x] Retención explícita y procedimiento de dead-letter (`scripts/dlq.py`).
- [x] Pruebas de regresión de los 10 criterios y de integración con Redis.

## Estado de cierre

Las pruebas con dobles verifican la lógica, no el servicio real. Antes de
declarar el alcance operativo cerrado faltan: la integración Redis en un Redis
real (CI o staging), la validación de estados de Meta con
`biz_opaque_callback_data` y el gate externo. Las limitaciones restantes están
en `docs/08_RECOVERY_AND_OPERATIONS.md`.

## Gate externo de activación

Preparación GCP (2026-10-20): workflow manual separado, referencias numéricas
de secretos, backend compartiendo clave sin cambiar autenticación, Direct VPC,
Redis AUTH/TLS con CA, preflight en Job, servicio HTTP y worker pool con drenaje.
Incluye pruebas offline; su despliegue, IAM/red reales e inferencia con la cuenta
runtime **siguen pendientes**. Runbook: `foods-infrastructure/docs/12_CONCIERGE_GCP.md`.

Antes de un go-live real se debe ejecutar, en el entorno autorizado, la validación con credenciales reales de Meta/Vertex AI/Redis/Fudia y una prueba de humo QR -> WhatsApp -> múltiples rondas -> KDS -> cuenta.

Ese gate requiere infraestructura y secretos externos. No se marca como ejecutado desde el repositorio y no representa código pendiente.
