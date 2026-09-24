# Arquitectura

Fudia Concierge es un servicio desplegable del monorepo Fudia.

```text
Meta WhatsApp
    |
    v
FastAPI webhook
    |
    v
Redis Stream durable
    |
    v
QueueWorker
    |
    v
ConciergeService
    |
    +--> scope classifier
    |
    +--> LangGraph intent router
           |
           +--> menu agent ----+
           +--> order agent ---+--> ToolRegistry --> foods-backend
           +--> service agent -+
    |
    v
WhatsApp outbound
```

## Fronteras

Concierge es responsable de conversación, intención, carrito temporal, presentación, confirmación de cada ronda y solicitud de cuenta.

foods-backend es responsable de tenancy, identidad del QR, menú, categorías, disponibilidad, precios, combos, stock, recetas, mesa, comanda acumulada, rondas de cocina, pagos, saldo, KDS y auditoría.

Una conversación mantiene un `conversationId` estable. Cada ronda confirmada usa un `requestId` distinto como clave idempotente. No se replica el catálogo comercial ni se calcula la cuenta desde memoria conversacional.

## Aislamiento y concurrencia

La identidad de sesión se deriva de:

`phone_number_id de Meta + teléfono del cliente`.

Esa identidad se hashea antes de formar la clave Redis. Esto evita que el mismo cliente colisione entre números/tenants distintos.

Cada conversación se serializa mediante un lock Redis renovable. Si otra entrega de la misma conversación llega mientras la primera sigue procesándose, espera el lock o se reintenta desde la cola. La sesión y el carrito se persisten únicamente bajo esa identidad compuesta.

## Cola durable

El webhook valida la firma de Meta y encola cada mensaje en Redis Streams. No ejecuta la conversación con `BackgroundTasks`.

La cola aplica:

- deduplicación atómica por `message_id`;
- consumer group;
- recuperación de mensajes pendientes mediante `XAUTOCLAIM`;
- ACK después de procesamiento y envío;
- reintentos;
- dead-letter al agotar intentos.

Cada worker limita sus mensajes en vuelo con `QUEUE_WORKER_CONCURRENCY`. La capacidad libre se pasa a Redis como límite de lectura, por lo que el proceso no reclama más entregas de las que puede ejecutar. Conversaciones distintas pueden avanzar en paralelo; los mensajes de una misma conversación siguen serializados por el lock distribuido renovable.

Durante el apagado, el worker deja de iterar trabajo nuevo y drena las entregas en vuelo dentro de `QUEUE_SHUTDOWN_GRACE_SECONDS`. Si se excede esa ventana, la tarea se cancela y Redis puede recuperar posteriormente cualquier entrega no confirmada.

El modelo de entrega es al menos una vez; las mutaciones de negocio críticas usan idempotencia en foods-backend.

## Orquestación multiagente

LangGraph enruta cada mensaje permitido hacia un especialista con privilegio mínimo:

- `menu`: solo tools de lectura de carta, combos y modificadores;
- `order`: tools de carta necesarias para resolver productos y mutaciones del carrito/pedido;
- `service`: únicamente cuenta y handoff humano.

El router de intención no responde al cliente. Cada agente dispone de su propio prompt y de un subconjunto de schemas. `ToolRegistry` vuelve a aplicar la misma autorización en código para impedir que un agente invoque una tool fuera de su dominio.

Los servicios de tools se separan en `MenuTools`, `OrderTools` y `ServiceTools`; `ConciergeTools` queda únicamente como fachada de compatibilidad.

## Resiliencia de dependencias

- timeouts explícitos para HTTP;
- backoff limitado;
- retries automáticos en GET;
- retry de creación de ronda únicamente porque `requestId` vuelve la operación idempotente;
- retries de envío WhatsApp;
- rate limiting por conversación;
- máximo de caracteres por mensaje;
- `/live` para proceso vivo;
- `/ready` para Redis/cola/sesiones y reachability de foods-backend.

## Configuración

En `ENV=production`, el servicio falla al arrancar si faltan secretos o parámetros críticos. Las dependencias directas están fijadas y `constraints.txt` congela el conjunto transitivo usado por CI y Docker.

## Observabilidad

El procesamiento crea un `traceId` a partir del `message_id` fingerprinted y lo propaga a foods-backend como `X-Correlation-ID`.

Se exponen métricas Prometheus de baja cardinalidad para:

- mensajes de webhook;
- decisiones de scope;
- routing por especialista;
- latencia y resultado de agentes;
- tools permitidas/denegadas;
- confirmación de pedido, cuenta y handoff;
- cola, retries y dead-letter;
- llamadas a foods-backend;
- envíos a WhatsApp.

Los logs no copian conversaciones, QR, credenciales ni teléfonos en claro.
