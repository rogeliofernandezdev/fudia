# Observabilidad — Fudia Concierge

## Principio

La observabilidad no debe convertir los logs ni las métricas en una copia de las conversaciones.

No se registran mensajes, tokens QR, credenciales ni números telefónicos en claro. Los identificadores externos se reducen a fingerprints SHA-256 truncados.

## Correlation tracing

Al procesar una entrega de la cola se genera un `traceId` determinista a partir del `message_id` fingerprinted.

Ese contexto acompaña:

- logs del worker;
- agente especialista;
- tools;
- llamada a foods-backend mediante `X-Correlation-ID`.

El valor no contiene el `message_id` original.

## Eventos estructurados

Concierge emite logs JSON para:

- entrega de cola completada;
- retry/dead-letter;
- agente completado;
- límite de tool calls;
- fallo del nivel Lite y si Flash tomó el turno (`concierge.agent.primary_failed`, `concierge.router.primary_failed`);
- aviso al cliente tras dead-letter, o su fallo (`concierge.queue.failure_notice_failed`);
- entrega aplazada sin consumir intento (`concierge.queue.deferred`: turno, lock o reconciliación);
- pérdida del lock o fallo al renovarlo (`concierge.lock.lost`, `concierge.lock.refresh_failed`);
- mensaje recuperado desde efectos persistidos (`concierge.processing.recovered`);
- mensaje o respuesta de un QR anterior (`concierge.message.superseded`, `concierge.outbound.superseded`);
- envío incierto o fallido (`concierge.outbound.uncertain`, `concierge.outbound.failed`);
- código corto de mesa no emitido al escanear; se usa el mensaje `FUDIA:<token>` (`concierge.start_code.failed`);
- predecesor ausente al ordenar (`concierge.queue.order_gap`);
- fallo de mantenimiento de cola (`concierge.queue.maintenance_failed`);
- errores operativos.

Para llamadas LLM se registran modelo, nivel (`lite`/`flash`), latencia, número de tools e input/output tokens (incluidos los de razonamiento), pero no prompt ni respuesta.

## Métricas Prometheus

`GET /metrics` expone métricas de baja cardinalidad:

- `fudia_concierge_inbound_messages_total`;
- `fudia_concierge_scope_decisions_total`;
- `fudia_concierge_intent_routes_total`;
- `fudia_concierge_agent_requests_total`;
- `fudia_concierge_agent_duration_seconds`;
- `fudia_concierge_tool_calls_total`;
- `fudia_concierge_business_actions_total`;
- `fudia_concierge_queue_deliveries_total`;
- `fudia_concierge_queue_duration_seconds`;
- `fudia_concierge_backend_requests_total`;
- `fudia_concierge_backend_duration_seconds`;
- `fudia_concierge_whatsapp_outbound_total`;
- `fudia_concierge_rate_limited_total`;
- `fudia_concierge_model_requests_total` (`role`, `tier`, `result`);
- `fudia_concierge_model_tokens_total` (`role`, `tier`, `direction`);
- `fudia_concierge_model_escalations_total` (`agent`, `result`);
- `fudia_concierge_lock_lost_total`;
- `fudia_concierge_processing_recoveries_total` (`effect`, `result`);
- `fudia_concierge_messages_superseded_total` (`phase`);
- `fudia_concierge_outbound_replies_total` (`state`);
- `fudia_concierge_queue_order_gaps_total`;
- `fudia_concierge_queue_trimmed_total` (`stream`);
- `fudia_concierge_queue_pending`, `fudia_concierge_queue_dead_letter`, `fudia_concierge_queue_stream_length` (estado compartido de Redis: agregar con `max`).

No se usan teléfonos, mesa, organización, local, QR, conversationId o messageId como labels.

## Trazabilidad persistente

foods-backend conserva la fuente auditable de acciones de negocio:

- pedidos creados o ampliados por Concierge;
- `conversationId` y `requestId`;
- solicitudes de handoff;
- usuario interno que resuelve handoff;
- solicitud de cuenta.

## Alertas recomendadas en infraestructura

El repositorio deja las señales listas; la plataforma de observabilidad del entorno debe materializar alertas sobre:

- crecimiento de dead-letter y de entradas pendientes;
- respuestas `uncertain`/`failed`, pérdidas de lock y huecos de orden (umbrales en `docs/08_RECOVERY_AND_OPERATIONS.md`);
- aumento de errores de foods-backend;
- fallos de WhatsApp;
- readiness en 503;
- p95/p99 de latencia;
- incremento anómalo de rate limiting;
- errores de modelo y escalados Lite → Flash por encima de lo habitual (cuotas o degradación de Vertex);
- consumo de tokens por nivel, para seguir el costo.

La definición del canal de alerta y su destino pertenece a infraestructura/despliegue y no se hardcodea en Concierge.

## Diagnóstico del preflight GCP

`python -m src.preflight` emite JSON con `event=concierge.preflight`, `stage`,
`status` (`started`, `passed`, `failed`) y `severity`. Las etapas distinguen
configuración/CA, Redis, salud del backend, clave de integración, acceso Meta,
cliente Vertex y cada nivel de modelo (`vertex_lite`, `vertex_flash`). Los
mensajes se vacían inmediatamente a stdout para conservar el último avance.

Un fallo incluye `reason` de una lista cerrada y una indicación operativa
`hint`. Cuando corresponde, añade solo el número `http_status` o
`tls_verify_code`. Clasifica también errores TLS envueltos por Redis/HTTP sin
copiar sus mensajes. No imprime excepciones completas, tracebacks, cuerpos de
respuesta, URLs, headers ni credenciales. Una causa desconocida se informa como
`unexpected_error` junto con la etapa; no se intenta adivinarla.

En Cloud Run revisar los registros de la ejecución del Job de preflight y buscar
`event=concierge.preflight`, `status=failed`. El proceso termina con código 1 y
no continúa a otras dependencias tras un fallo. El diagnóstico no desactiva TLS,
no modifica IAM, no consume mensajes ni crea pedidos. Su resultado no sustituye
la prueba de activación externa.

La comprobación Redis usa solo `PING`, `INFO memory` (`maxmemory_policy`) e
`INFO cluster` (`cluster_enabled`). No usa `CONFIG GET`: Memorystore bloquea
`CONFIG` incluso para lecturas. La política debe ser explícitamente `noeviction`
y cluster debe informar `0`; una respuesta incompleta también bloquea el release.
No se requieren permisos IAM adicionales ni cambios de configuración Redis para
hacer estas consultas. Referencias:
[comandos de Memorystore](https://docs.cloud.google.com/memorystore/docs/redis/supported-commands)
y [campos de INFO](https://redis.io/docs/latest/commands/info/).
