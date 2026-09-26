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
- errores operativos.

Para llamadas LLM se registran modelo, latencia, número de tools e input/output tokens, pero no prompt ni respuesta.

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
- `fudia_concierge_rate_limited_total`.

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

- crecimiento de dead-letter;
- aumento de errores de foods-backend;
- fallos de WhatsApp;
- readiness en 503;
- p95/p99 de latencia;
- incremento anómalo de rate limiting.

La definición del canal de alerta y su destino pertenece a infraestructura/despliegue y no se hardcodea en Concierge.
