# Plan de implementación

## Bloque 1 — Fundación

- [x] Proyecto en monorepo.
- [x] FastAPI, liveness y readiness.
- [x] Webhook WhatsApp con firma.
- [x] Adaptador WhatsApp.
- [x] OpenAI Responses API con tools.
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

- [x] Número WhatsApp configurable por empresa/local.
- [x] Botón Pedir por WhatsApp en página QR.
- [x] Deeplink con token.
- [x] Activar/desactivar Concierge por empresa/local.
- [x] Contrato server-to-server bloqueado cuando el local desactiva Concierge.
- [x] Sesión conversacional rechaza QR de local desactivado.
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

## Estado de cierre

**Implementación de repositorio: 100% para el alcance definido de Fudia Concierge.**

No quedan ❌ ni ⚠️ de implementación conocidos en este alcance después de la puerta de CI.

## Gate externo de activación

Antes de un go-live real se debe ejecutar, en el entorno autorizado, la validación con credenciales reales de Meta/OpenAI/Redis/Fudia y una prueba de humo QR -> WhatsApp -> múltiples rondas -> KDS -> cuenta.

Ese gate requiere infraestructura y secretos externos. No se marca como ejecutado desde el repositorio y no representa código pendiente.
