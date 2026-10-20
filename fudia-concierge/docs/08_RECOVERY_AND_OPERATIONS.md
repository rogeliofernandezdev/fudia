# Recuperación, consistencia y operación

## Garantías

Concierge procesa cada mensaje **al menos una vez** y está diseñado para que
repetirlo no repita acciones:

- un mismo `message_id` nunca ejecuta dos veces el modelo después de haber
  persistido un efecto o una respuesta;
- toda operación comercial usa una clave idempotente estable que se guarda
  **antes** de llamar a foods-backend;
- si falla el envío a WhatsApp solo se reenvía la respuesta ya preparada;
- un worker que perdió el lock de la conversación no puede escribir estado;
- los mensajes de una conversación se procesan en el orden en que el webhook
  los aceptó; conversaciones distintas avanzan en paralelo.

No se promete entrega exactamente una vez en WhatsApp: ante un envío incierto
se reconcilia y, si no hay confirmación, el texto puede repetirse una vez.
Nunca se repite una acción comercial.

## Ciclo de un mensaje

```text
webhook ──XADD (dedup + seq)──> Stream
                                  │
worker: espera su turno (seq) ────┤
        fase 1: process_inbound ──┼─> registro de procesamiento + sesión (Redis)
        fase 2: ReplyDispatcher ──┼─> WhatsApp (biz_opaque_callback_data)
        ACK (libera el turno) <───┘
webhook de estados de Meta ──> fudia:concierge:outbound:{ref}
```

## Registro de procesamiento

Clave `fudia:concierge:processing:{sha256(message_id)}`. Contiene:

| Campo | Uso |
| --- | --- |
| `message_id` | Identificador original de Meta. |
| `conversation` | Hash de `phone_number_id + cliente`. |
| `conversation_id` | Contexto QR al que pertenece la respuesta. |
| `stage` | Etapa actual (tabla siguiente). |
| `attempts`, `send_attempts`, `resends` | Intentos de proceso, envío y reenvío. |
| `effects[]` | Cambios realizados: tipo, estado, `requestId`, datos de reintento y resultado mínimo. |
| `reply` | Respuesta preparada. |
| `last_send_at`, `created_at`, `updated_at` | Fechas. |
| `last_error` | Solo tipo de error; nunca texto, teléfono, QR ni URL. |

Los datos de reintento de una operación (`payload`) incluyen el token QR y el
teléfono porque son necesarios para repetirla con la misma clave. Viven solo
en Redis privado, igual que la sesión, y nunca se registran en logs.

### Etapas y qué hace un reintento

| Etapa | Significado | Reintento |
| --- | --- | --- |
| `processing` | Comenzó; ningún efecto persistido. | Ejecuta el turno de nuevo (seguro). |
| `effects_applied` | Se persistió al menos un cambio de carrito, confirmación u operación. | No llama al modelo: responde desde los efectos guardados. |
| `effect_pending` | Se guardó la clave de una operación del backend, pero no su resultado. | Repite la operación con la **misma** clave y responde desde su resultado. |
| `reply_ready` | Respuesta lista, sin enviar. | Solo envía. |
| `sending` / `uncertain` | Envío iniciado o sin respuesta de Meta. | Reconciliación (ver Envío). |
| `sent` | Meta aceptó el mensaje. | Confirma la entrega sin reenviar. |
| `failed` | Envío no confirmado tras el reenvío permitido, o Meta informó fallo. | Confirma la entrega sin reenviar. |
| `superseded` | Pertenece a un QR anterior. | No se aplica ni se envía. |

### Puntos de recuperación

Cada punto escribe sesión y registro **juntos** mediante un script Lua que
antes verifica el lock (`RedisConversationStore.commit`):

1. inicio del intento (`processing`, sin sesión);
2. contexto QR validado (junto con la respuesta de bienvenida);
3. cada cambio del carrito (`cart`), de los datos de entrega (`delivery`) y la confirmación preparada (`confirmation`);
4. antes de crear una ronda, solicitar handoff o pedir la cuenta (`effect_pending` con clave y datos de reintento);
5. resultado de esa operación (`effects_applied`);
6. respuesta lista para enviar (`reply_ready`).

### Idempotencia por operación

| Operación | Protección |
| --- | --- |
| Agregar/quitar productos | El registro impide volver a ejecutar el turno; el carrito se guarda con el efecto. |
| Preparar confirmación | `requestId` se crea y persiste en la sesión al preparar. |
| Crear ronda | Se reutiliza `pending_order_request_id`; si no existía, se crea y persiste antes de llamar. foods-backend devuelve la misma ronda ante el mismo `requestId`. |
| Crear pedido delivery/recojo | Igual que una ronda: `requestId` persistido antes de llamar y cuerpo completo guardado como datos de reintento (`mode=delivery`). foods-backend devuelve el mismo pedido ante el mismo `requestId` aunque la tarifa haya cambiado; `delivery_fee_changed` (4xx) marca el efecto `failed`, recotiza e invalida la confirmación. |
| Handoff | foods-backend mantiene un único handoff pendiente por conversación (índice único). |
| Cuenta | `requestId` nuevo por solicitud, persistido antes de llamar. foods-backend registra cada `requestId` una sola vez (índice único `audit_log_concierge_bill_request_uq`) y devuelve la cuenta vigente al repetirlo; por eso la llamada también se reintenta ante 5xx. |

Un error 4xx del backend marca el efecto `failed` (no ocurrió). Un error 5xx o
de red lo marca `unknown`: el `requestId` permanece en la sesión y el siguiente
"sí" del cliente recupera la misma ronda.

## Lock de conversación con fencing

- Lock Redis con token propio y TTL (`CONVERSATION_LOCK_TTL_SECONDS`),
  renovado cada TTL/6.
- Si la renovación devuelve que el token ya no es nuestro, o falla por más del
  80 % del TTL, el lease se marca perdido y **se cancela la tarea protegida**:
  no continúan llamadas al modelo, tools ni escrituras.
- Toda escritura de sesión/registro es condicional al token (Lua).
  Un worker antiguo recibe `LockLostError` y no sobrescribe al nuevo.
- Antes de cada operación del backend se comprueba de nuevo la propiedad.
  Si el lease se pierde durante la llamada, el registro queda en
  `effect_pending` y el reintento reconcilia con la misma clave.
- `LockLostError` y la espera de lock agotada no cuentan como intento: la
  entrada se queda pendiente en el Stream con el mismo id.

## Orden por conversación

Un lock evita simultaneidad, pero no ordena. Por eso el webhook asigna en el
mismo script Lua del `XADD` una secuencia por conversación
(`fudia:concierge:seq:{conversation}`). El worker procesa la secuencia `n`
solo cuando `fudia:concierge:seqdone:{conversation} >= n-1`; el ACK o el
dead-letter avanzan ese valor.

- Los reintentos conservan secuencia, `message_id` y conversación.
- Si la espera supera `QUEUE_TURN_WAIT_SECONDS`, la entrega se aplaza sin
  copiarla (queda pendiente y `XAUTOCLAIM` la retoma).
- Si el predecesor falta más de `QUEUE_ORDER_MAX_WAIT_SECONDS`, se procesa
  igualmente y se cuenta en `fudia_concierge_queue_order_gaps_total`.
- La renovación de visibilidad (`touch`) solo actúa si este consumidor sigue
  siendo dueño de la entrada; no se la quita a otro worker que la recuperó.

## Cambio de QR

- El QR abre un contexto nuevo: `conversationId` nuevo, carrito, confirmación,
  `requestId` y handoff pendiente reiniciados.
- La sesión guarda la secuencia del mensaje QR (`context_sequence`). Un mensaje
  encolado antes (por ejemplo, reencolado desde dead-letter) queda
  `superseded` y no toca la nueva mesa.
- Cada respuesta guarda su `conversation_id`; si al enviarla el contexto ya es
  otro, se marca `superseded` y no se envía.

## Envío y reconciliación con Meta

1. `reply_ready → sending` (persistido antes de llamar a Meta).
2. Respuesta 2xx → `sent`.
3. Error de conexión, 429 o 5xx → Meta no aceptó: vuelve a `reply_ready` y la
   entrega se reintenta con la misma respuesta.
4. Timeout o conexión cortada después de escribir la solicitud → `uncertain`.
   El envío lleva `biz_opaque_callback_data = sha256(message_id)`; el webhook
   registra los estados `sent/delivered/read/failed` que Meta devuelve con esa
   referencia.
5. Durante `OUTBOUND_RECONCILE_SECONDS` se espera ese estado. Si llega
   confirmado → `sent`; si llega `failed` → `failed`; si no llega nada, se
   reenvía hasta `OUTBOUND_MAX_RESENDS` veces (1 por defecto); si tampoco se
   confirma, `failed` con `delivery_unconfirmed`.

`OUTBOUND_MAX_RESENDS` es una decisión de producto, porque la API de envío de
Meta no acepta clave de idempotencia y no hay forma de saber si un envío sin
respuesta llegó:

| Valor | Efecto si Meta nunca confirma |
| --- | --- |
| `1` (defecto) | El cliente recibe respuesta; el texto puede llegar dos veces. |
| `0` | El texto nunca se duplica; el cliente puede quedarse sin respuesta. |

Nunca se repite una acción comercial con ninguno de los dos valores.

Un `sending` que quedó así por un reinicio se trata igual que `uncertain`.

## Checkpoints de LangGraph: decisión

Se evaluó `langgraph-checkpoint-redis` y no se adopta:

- las tools se ejecutan dentro del bucle de function calling del agente, no
  como nodos del grafo; un checkpoint por nodo no registraría los efectos
  ocurridos a mitad del turno;
- según su documentación requiere capacidades de búsqueda y JSON de Redis
  (Redis Stack/Redis 8), cuya disponibilidad en Memorystore debe confirmarse;
- un checkpoint no aporta idempotencia: las operaciones con efectos necesitan
  su propia clave.

El registro de procesamiento cubre exactamente los puntos necesarios con
escrituras condicionales al lock. Si el grafo pasa a ejecutar tools como
nodos, revisar esta decisión.

## Retención

| Dato | Retención | Regla |
| --- | --- | --- |
| Entradas del Stream | `QUEUE_RETENTION_SECONDS` (24 h) | `XTRIM MINID` hasta el menor entre el corte, la entrada pendiente más antigua y la última entregada al grupo. Nunca borra pendientes ni no leídas. |
| Dead-letter | `QUEUE_DEAD_LETTER_RETENTION_SECONDS` (14 d) | `XTRIM MINID` por antigüedad. |
| Deduplicación | 24 h (`queued`/`completed`), 5 min tras dead-letter | Permite que Meta reintente un mensaje que fue a dead-letter. |
| Registros terminados (`sent`, `failed`, `superseded`) | `PROCESSING_RECORD_TTL_SECONDS` (3 d) | |
| Registros recuperables (cualquier otra etapa) | `PROCESSING_PENDING_RECORD_TTL_SECONDS` (14 d) | Debe cubrir la retención del dead-letter (validado al arrancar). |
| Secuencias por conversación | `QUEUE_ORDER_TTL_SECONDS` (7 d) | Debe superar el TTL de sesión (validado al arrancar). |
| Sesión/carrito | `SESSION_TTL_SECONDS` (12 h) | |
| Memoria del cliente (último local por QR y nombre) | `CUSTOMER_MEMORY_TTL_SECONDS` (90 d) | Clave `fudia:concierge:customer:{identity}`, escrita en el mismo commit con fencing que la sesión. Permite pedidos de recojo/delivery cuando la sesión expiró. |
| Estados de Meta | 24 h | |

Confirmar (`XACK`) no borra la entrada: solo la retención lo hace. La tarea de
mantenimiento del worker aplica la retención cada
`QUEUE_MAINTENANCE_INTERVAL_SECONDS` y actualiza los gauges de la cola.

## Procedimiento de dead-letter

```bash
python scripts/dlq.py list
python scripts/dlq.py show <entry_id>
python scripts/dlq.py requeue <entry_id> --yes
python scripts/dlq.py discard <entry_id> --yes
```

El script muestra solo fingerprints, intentos, tipo de error, etapa y el
siguiente paso que ocurrirá; nunca teléfonos, textos, QR ni respuestas.

1. Corregir la causa (dependencia caída, configuración, error de código).
2. `show` para revisar la etapa. `reply_ready`/`uncertain` solo enviará;
   `effect_pending` repetirá la operación con su clave original;
   `processing` o sin registro volverá a ejecutar el modelo.
3. `requeue` conserva `message_id` y secuencia. Si el cliente escaneó otro QR
   después, el mensaje queda `superseded`.
4. `discard` solo cuando el mensaje ya no debe atenderse.

## Despliegue en GCP

Implementación del release automático y alternativa manual, con pasos exactos:
[Concierge GCP](../../foods-infrastructure/docs/12_CONCIERGE_GCP.md).
El coordinador `foods-gcp` lo invoca por push a `feat/ui-standard` si Concierge
está pendiente. Verifica y ejecuta un Job de preflight privado antes de HTTP/worker
pool; exige backend activo con la clave compartida, sin forzar su despliegue.
La opción manual permite desplegar también backend con confirmación explícita.
`REDIS_CA_CERTFILE` conecta todos los clientes y DLQ con CA y verificación TLS,
sin overrides inseguros en URI. El worker maneja SIGTERM mediante el lifespan
y drenaje acotado. Default del release GCP: sesión 3600 s, sin cambiar los
defaults locales/retenciones de recuperación. Un push relevante en esa rama sí
puede activar el bot; preparar variables/IAM antes. Un release parcial queda
pendiente hasta que HTTP y la revisión activa del worker completen el mismo SHA.

Un solo proyecto y una sola imagen con dos roles:

| Rol | Ejecución | Escalado |
| --- | --- | --- |
| HTTP ingress | Cloud Run service, `WORKER_ENABLED=false`. Valida firma y encola; no ejecuta IA. | Por solicitudes. |
| Worker | `python -m src.worker`. Cloud Run worker pool (confirmar disponibilidad en la región) o Cloud Run service con CPU siempre asignada (`--no-cpu-throttling`) y `min-instances>=1`. | Independiente; `QUEUE_WORKER_CONCURRENCY` por instancia. |

Nunca depender de una tarea en background de un servicio con CPU asignada
solo durante solicitudes.

**Redis (Memorystore):**

- acceso solo por IP privada (Direct VPC egress o conector), AUTH y TLS en tránsito;
- `maxmemory-policy noeviction`: una expulsión podría borrar el Stream, locks,
  secuencias o registros recuperables. Con `noeviction` Redis rechaza
  escrituras al llenarse y el webhook responde error para que Meta reintente;
- persistencia activada (AOF si la variante lo ofrece; si solo hay RDB, se
  puede perder lo escrito desde la última instantánea);
- **modo sin cluster**: los scripts Lua usan varias claves que no comparten
  hash slot.

**Identidades y secretos:** cuenta de servicio propia por rol, sin claves
JSON. Worker: `roles/aiplatform.user` (Vertex) y acceso solo a sus secretos
(Meta, Redis, `FUDIA_CONCIERGE_API_KEY`). Ingress: solo secretos de Meta y Redis.

**Apagado:** Cloud Run da unos 10 s tras SIGTERM. `QUEUE_SHUTDOWN_GRACE_SECONDS=8`
deja de leer y drena; lo que no termina queda pendiente y otro worker lo
recupera con su registro.

**Métricas y alertas:** recoger `/metrics` de todas las réplicas con un
colector central (Managed Service for Prometheus u OpenTelemetry). Los gauges
de cola reflejan el estado compartido de Redis: agregar con `max`, no `sum`.
Alertas mínimas:

| Señal | Condición sugerida |
| --- | --- |
| `fudia_concierge_queue_pending` | crecimiento sostenido 10 min |
| `fudia_concierge_queue_dead_letter` | > 0 nuevo |
| `fudia_concierge_outbound_replies_total{state="uncertain"|"failed"}` | tasa > 0 sostenida |
| `fudia_concierge_lock_lost_total` | > 0 |
| `fudia_concierge_queue_order_gaps_total` | > 0 |
| `fudia_concierge_processing_recoveries_total` | aumento anómalo |
| `fudia_concierge_queue_duration_seconds` | p95 > objetivo |
| `fudia_concierge_backend_requests_total{result="error"}`, `whatsapp_outbound_total{result!="success"}`, `model_requests_total{result="error"}` | tasa sobre línea base |

No se crean recursos pagados, se cambia tráfico ni se despliega desde este
repositorio sin autorización explícita.

## Pruebas

| Nivel | Cómo | Cubre |
| --- | --- | --- |
| Unitarias | `pytest -q` (`tests/test_recovery.py`) | Los 10 criterios de aceptación con store, lock y cola en memoria. |
| Integración Redis | `FUDIA_REDIS_INTEGRATION=1 REDIS_URL=... pytest -q tests/test_redis_integration.py` | Commit condicionado, `touch` sin robo, orden y reintentos, retención, requeue de dead-letter, worker de punta a punta. |
| Carga Redis | `FUDIA_REDIS_LOAD_TEST=1 REDIS_URL=... pytest -q tests/test_capacity_redis.py` | Throughput de Streams. |
| Externa | Entorno autorizado | Meta (firma, envío, estados con `biz_opaque_callback_data`), Vertex/OpenAI y foods-backend real. |

## Limitaciones

- La API de envío de Meta no es idempotente: tras un envío incierto sin estado
  de Meta, el texto puede llegar dos veces (`OUTBOUND_MAX_RESENDS=1`) o no
  llegar (`OUTBOUND_MAX_RESENDS=0`). Es una propiedad de Meta, no del código.
- La reconciliación depende de que Meta devuelva `biz_opaque_callback_data`
  en los estados; debe validarse con la cuenta real.
- La idempotencia de la cuenta exige la migración
  `000082_concierge_bill_request_idempotency` de foods-backend. Sin ella, el
  backend acepta `requestId` pero no evita el registro duplicado.
- Si el cliente escribe a mitad de un envío incierto, su siguiente mensaje
  espera hasta `OUTBOUND_RECONCILE_SECONDS` por el orden de la conversación.
- Un mensaje `superseded` no recibe respuesta.
- Requisito de despliegue, no defecto: Redis sin cluster (los scripts Lua
  combinan el Stream global con claves por conversación; soportar cluster
  exigiría repartir la cola en varios Streams) y con `noeviction` (cualquier
  otra política permitiría a Redis borrar mensajes pendientes o locks).
- La integración Redis se ejecutó localmente con un emulador (fakeredis con
  Lua). El job `capacity` de CI la ejecuta contra Redis 7 real; la versión de
  Memorystore elegida debe validarse en staging.
