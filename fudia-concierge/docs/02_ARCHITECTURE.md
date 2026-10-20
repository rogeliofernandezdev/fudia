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
QueueWorker (turno por conversación)
    |
    v
ConciergeService (fase 1: procesar y persistir)
    |
    +--> LangGraph unified router
           |
           +--> out_of_scope
           +--> menu agent ----+
           +--> order agent ---+--> ToolRegistry --> foods-backend
           +--> service agent -+
    |
    v
ReplyDispatcher (fase 2: enviar la respuesta persistida)
    |
    v
WhatsApp outbound
```

## Fronteras

Concierge es responsable de conversación, intención, carrito temporal, presentación, confirmación de cada ronda y solicitud de cuenta.

foods-backend es responsable de tenancy, identidad del QR, menú, categorías, disponibilidad, precios, combos, stock, recetas, mesa, comanda acumulada, rondas de cocina, pagos, saldo, KDS y auditoría.

Una conversación mantiene un `conversationId` estable. Cada ronda confirmada usa un `requestId` distinto como clave idempotente. No se replica el catálogo comercial ni se calcula la cuenta desde memoria conversacional.

## Canal WhatsApp global

Fudia opera un único bot de WhatsApp para todas las empresas. El número oficial se administra exclusivamente en Meta / WhatsApp Business; no se persiste ni se configura manualmente en foods-backend, Admin Web ni en variables propias de negocio.

`WHATSAPP_PHONE_ID` identifica técnicamente el activo de Meta usado por Concierge. Cuando un comensal pulsa “Pedir por WhatsApp”, Admin Web delega en `/start/{qr}`; Concierge valida el QR y el entitlement `organization_modules.whatsapp_bot`, consulta a Meta el `display_phone_number` asociado al Phone Number ID y genera el deeplink `wa.me`. La respuesta de Meta se mantiene solo en caché temporal.

El QR de mesa contiene un token opaco que permite a foods-backend resolver `organization_id`, `location_id` y `table_id`. La activación de `whatsapp_bot` pertenece exclusivamente a la administración global de plataforma.

Como todos los restaurantes comparten el mismo chat oficial, escanear un nuevo QR reemplaza el contexto conversacional activo del cliente por la nueva empresa/local/mesa y reinicia el carrito temporal de Concierge.

## Aislamiento y concurrencia

La identidad de sesión se deriva de:

`phone_number_id de Meta + teléfono del cliente`.

Esa identidad se hashea antes de formar la clave Redis. El tenant no se deriva del número de WhatsApp: siempre lo determina el QR activo de la conversación.

Cada conversación se serializa mediante un lock Redis renovable con fencing: si el lease se pierde, el worker cancela el trabajo protegido y toda escritura de sesión o registro se rechaza porque se valida contra el token del lock dentro de un script Lua. El orden lo garantiza además una secuencia por conversación asignada al encolar. La sesión y el carrito se persisten únicamente bajo esa identidad compuesta. Detalle en `docs/08_RECOVERY_AND_OPERATIONS.md`.

## Cola durable

El webhook valida la firma de Meta y encola cada mensaje en Redis Streams. No ejecuta la conversación con `BackgroundTasks`.

La cola aplica:

- deduplicación atómica por `message_id`;
- secuencia por conversación asignada en el mismo `XADD`;
- consumer group;
- recuperación de mensajes pendientes mediante `XAUTOCLAIM`;
- renovación de visibilidad solo mientras el consumidor sigue siendo dueño;
- ACK después de procesamiento y envío;
- reintentos que conservan `message_id` y secuencia;
- aplazamiento sin copia cuando falta el turno, el lock o la confirmación de Meta;
- dead-letter al agotar intentos, con procedimiento de recuperación (`scripts/dlq.py`);
- retención que nunca recorta entradas pendientes ni no leídas.

Cada worker limita sus mensajes en vuelo con `QUEUE_WORKER_CONCURRENCY`. La capacidad libre se pasa a Redis como límite de lectura, por lo que el proceso no reclama más entregas de las que puede ejecutar. Mientras una entrega permanece activa —incluido el tiempo esperando el lock de conversación— el worker renueva periódicamente su idle time en Redis para que `XAUTOCLAIM` no la recupere prematuramente. Conversaciones distintas pueden avanzar en paralelo; los mensajes de una misma conversación siguen serializados por el lock distribuido renovable.

Durante el apagado, el worker deja de iterar trabajo nuevo y drena las entregas en vuelo dentro de `QUEUE_SHUTDOWN_GRACE_SECONDS`. Si se excede esa ventana, la tarea se cancela y Redis puede recuperar posteriormente cualquier entrega no confirmada.

El modelo de entrega es al menos una vez. Cada mensaje tiene un registro de procesamiento en Redis: una vez persistido un efecto o una respuesta, un reintento no vuelve a ejecutar el modelo; responde desde lo guardado, repite la operación pendiente con su misma clave idempotente o solo reenvía la respuesta. Ver `docs/08_RECOVERY_AND_OPERATIONS.md`.

## Orquestación multiagente

LangGraph usa una única decisión de routing por mensaje: `out_of_scope`, `menu`, `order` o `service`. `out_of_scope` termina el flujo sin invocar un agente; las tres rutas operativas llegan a un especialista con privilegio mínimo:

- `menu`: solo tools de lectura de carta, combos y modificadores;
- `order`: tools de carta necesarias para resolver productos y mutaciones del carrito/pedido;
- `service`: únicamente cuenta y handoff humano.

El router no responde al cliente y cualquier salida inválida se trata como `out_of_scope`. Cada agente dispone de su propio prompt y de un subconjunto de schemas. `ToolRegistry` vuelve a aplicar la misma autorización en código para impedir que un agente invoque una tool fuera de su dominio.

Los servicios de tools se separan en `MenuTools`, `OrderTools` y `ServiceTools`; `ConciergeTools` queda únicamente como fachada de compatibilidad.

### Modelos por dificultad

El proveedor por defecto es Gemini en Vertex AI. El router, `menu` y `service` usan Flash-Lite; `order` usa Flash. Si un turno Flash-Lite falla antes de ejecutar una tool con efectos, Flash lo repite una vez; el router Lite también cae a Flash ante errores. Detalle, configuración y requisitos de GCP en `docs/07_MODEL_PROVIDERS.md`.

## Resiliencia de dependencias

- timeouts explícitos para HTTP;
- backoff limitado;
- retries automáticos en GET;
- retry de creación de ronda únicamente porque `requestId` vuelve la operación idempotente;
- retries de envío WhatsApp;
- timeout por llamada al modelo, reintentos del SDK ante 408/429/5xx y topes de salida y de vueltas de tools;
- escalado Flash-Lite → Flash sin repetir acciones con efectos;
- aviso al cliente cuando su mensaje agota los reintentos y pasa a dead-letter;
- rate limiting por conversación;
- máximo de caracteres por mensaje;
- `/live` para proceso vivo;
- `/ready` para Redis/cola/sesiones y reachability de foods-backend.

## Configuración

El factory de composición está en `src/application.py`. `src/main.py` crea la
app ASGI; el worker importa solo el factory para no construir una app/cliente
LLM HTTP adicional al iniciar. `REDIS_CA_CERTFILE` aplica un único bundle CA y
TLS verificado a todos los clientes Redis; mantenerlo vacío solo en desarrollo
local si Redis no requiere CA privada. Release GCP y preflight en
`foods-infrastructure/docs/12_CONCIERGE_GCP.md`.

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
- envíos a WhatsApp;
- llamadas, tokens y escalados de modelo por rol y nivel (`fudia_concierge_model_*`).

Los logs no copian conversaciones, QR, credenciales ni teléfonos en claro.

## Independencia de despliegue y extracción futura

Concierge **ya es un servicio independiente del ejecutable Go**: el chatbot no
importa paquetes de `foods-backend` ni accede directamente a PostgreSQL.
Intercambia datos a través de contratos HTTP versionables y
`X-Fudia-Concierge-Key`. El dominio, los servicios y la orquestación no
pueden importar adaptadores concretos; este límite está protegido por
`tests/test_architecture_guards.py` y `src/services/ports.py`.

La independencia significa **autonomía de código, contenedor, ciclo de despliegue
y escalado**; no significa operar sin dependencias externas. El bot necesita
Meta WhatsApp Business Cloud API para recibir/enviar, Redis para estados/cola,
un proveedor LLM para clasificación, y la API FUDIA para catálogo/pedidos.
Un fallo externo sigue afectando la experiencia; los reintentos, timeouts y
DLQ limitan efectos, pero no garantizan disponibilidad completa.

### Dos roles de un mismo código y Docker

- **HTTP ingress:** establecer `WORKER_ENABLED=false`, iniciar
  `uvicorn src.main:app --host 0.0.0.0 --port 8010`. Atiende únicamente
  `/start/{qr}`, `/webhook`, salud y métricas. Tras autenticar la firma
  de Meta, registra el mensaje en Redis y responde sin ejecutar IA.
- **Worker:** ejecutar `python -m src.worker` con credenciales, Redis y
  `QUEUE_WORKER_CONCURRENCY`. Consume Streams, controla concurrencia,
  locks distribuidos y resultados de Meta. Puede tener réplicas independientes
  del servicio HTTP; todas deben compartir el mismo Redis y grupo de consumo.

Por retrocompatibilidad, `WORKER_ENABLED=true` continúa siendo el modo
**combinado** en desarrollo y Docker actual. Para Cloud Run con asignación de
CPU solo durante solicitudes HTTP, **no iniciar workers de larga vida en el
servidor webhook**: usar un worker pool o infraestructura que garantice CPU
y procesos activos permanentemente. Verificar esa modalidad y el costo antes
de activar producción. No presupone autoscaling ya configurado.

### Limitaciones de consistencia que exigen diseño adicional

- Streams procesan **al menos una vez**. Las acciones comerciales no se
  repiten (registro de procesamiento + `requestId`), pero el envío WhatsApp
  sigue siendo al menos una vez: un envío incierto se reconcilia con los
  estados de Meta y, sin confirmación, se reenvía según `OUTBOUND_MAX_RESENDS`.
- Las métricas Prometheus se exponen por **instancia**; con réplicas se
  necesita colector central. Los gauges de cola reflejan Redis compartido y
  se agregan con `max`. Alertas en `docs/08_RECOVERY_AND_OPERATIONS.md`.
- El código no demuestra aún throughput de negocio real, tiempo p95/p99
  bajo carga sostenida, cumplimiento de rate limits de Meta, ni resistencia
  de múltiples réplicas con caídas simultáneas. Estas pruebas precisan Redis
  real, un simulador de proveedores y después validación externa autorizada.
- `/ready` comprueba Redis y exige HTTP 200 de `foods-backend/health`.
  El webhook de Meta también verifica HMAC SHA-256 de bytes originales,
  limita el tamaño de solicitudes y acepta texto, botones y listas interactivos.
- En `ENV=production` los secretos Meta, Redis y FUDIA son obligatorios,
  junto con `GOOGLE_CLOUD_PROJECT` (Vertex) u `OPENAI_API_KEY` (OpenAI). Publicar Redis solo en red privada y gestionar secretos en
  Secret Manager o equivalente.

### Pruebas de activación externa

No afirmar disponibilidad productiva ni comunicación real de WhatsApp sin
validar firmas y desafíos desde Meta, webhook HTTPS público, permisos del
número oficial, templates aprobados cuando apliquen, ventanas de mensajería,
rotación de tokens y prueba punta a punta con una empresa/mesa de prueba.
