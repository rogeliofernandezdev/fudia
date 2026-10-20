# Fudia Concierge

Asistente conversacional de Fudia para atender por WhatsApp a un comensal sentado en una mesa, desde el escaneo del QR hasta la solicitud de la cuenta.

## Flujo

QR de mesa -> endpoint de inicio -> Meta resuelve el número oficial desde `WHATSAPP_PHONE_ID` -> WhatsApp -> webhook Meta -> Redis Stream durable -> worker -> router unificado LangGraph -> agente especialista -> tools permitidas -> foods-backend -> respuesta WhatsApp.

Fudia Concierge mantiene únicamente el estado conversacional y el carrito temporal de la ronda en curso. foods-backend revalida producto, disponibilidad, precio, mesa y stock antes de registrar cada ronda y conserva la comanda acumulada como fuente de verdad.

Cuando el cliente solicita la cuenta, Concierge consulta el consumo real de la mesa en backend y muestra productos, total, pagos registrados y saldo pendiente. Solicitar la cuenta no registra un pago; el cobro continúa por Caja/POS.

## Arquitectura multiagente

Cada mensaje se clasifica en una sola llamada como `out_of_scope`, `menu`, `order` o `service`. Los mensajes válidos se enrutan a uno de tres especialistas:

- `menu`: lectura de carta, categorías, combos y modificadores;
- `order`: carrito, configuración, confirmación y envío de rondas;
- `service`: cuenta y handoff humano.

Cada especialista recibe únicamente sus schemas de tools y `ToolRegistry` vuelve a aplicar el mismo límite en código. El prompt no constituye una frontera de seguridad.

## Resiliencia

- Redis Streams para entrega durable del webhook.
- Concurrencia acotada por worker con backpressure antes de reclamar mensajes.
- Heartbeat de visibilidad solo mientras el consumidor sigue siendo dueño de la entrega.
- Procesamiento paralelo de conversaciones distintas y orden por conversación mediante secuencia asignada al encolar.
- Deduplicación atómica por `message_id`.
- Registro de procesamiento por mensaje: un reintento no vuelve a ejecutar el modelo tras persistir efectos o respuesta.
- Dos fases: procesar y persistir; después enviar la respuesta guardada. ACK solo al cerrar el envío.
- Envíos inciertos a WhatsApp reconciliados con estados de Meta (`biz_opaque_callback_data`).
- Reintentos, dead-letter y recuperación con `scripts/dlq.py`.
- Lock distribuido renovable con fencing: perder el lease cancela el trabajo y bloquea escrituras.
- Retención explícita de Stream, dead-letter y registros sin borrar pendientes.
- Sesiones aisladas por `phone_number_id` de Meta + cliente.
- Rate limiting por conversación.
- Límite de tamaño de mensaje.
- Retries con backoff para lecturas de backend, creación idempotente de rondas y envío WhatsApp.
- `/live` para liveness y `/ready` para dependencias críticas.

## Observabilidad

- logs JSON sin conversaciones, QR ni teléfonos en claro;
- `traceId` correlacionable desde cola hasta backend;
- métricas Prometheus en `/metrics`;
- métricas de cola, mensajes en vuelo, agentes, tools, negocio, backend y WhatsApp;
- fingerprints SHA-256 truncados para identificadores externos en logs;
- eval corpus para alcance, routing e intentos de prompt injection.

## Stack

- Python 3.12
- FastAPI
- LangGraph
- Gemini en Vertex AI (`google-genai`) con function calling manual; Flash-Lite y Flash por dificultad
- OpenAI Responses API como proveedor alternativo (`LLM_PROVIDER=openai`)
- WhatsApp Business Cloud API
- Redis para sesión, locks, rate limiting y cola durable
- Prometheus client
- HTTP para integración con foods-backend

## Prompts

- `prompts/concierge_router.md`: clasificación unificada `out_of_scope / menu / order / service`.
- `prompts/system.md`: reglas comunes.
- `prompts/menu_agent.md`: especialista de carta.
- `prompts/order_agent.md`: especialista de pedido.
- `prompts/service_agent.md`: especialista de servicio.

Las reglas de negocio críticas no viven en prompts: se validan en services/tools y foods-backend.

## Desarrollo local

1. Copiar `.env.example` a `.env`.
2. Configurar Vertex AI (`GOOGLE_CLOUD_PROJECT` y `gcloud auth application-default login`), Redis y credenciales de Meta/WhatsApp. Ver `docs/07_MODEL_PROVIDERS.md`. `WHATSAPP_PHONE_ID` es el identificador técnico del activo en Meta, no el número telefónico. El número visible se obtiene desde Meta y no se configura en Fudia.
3. Instalar de forma reproducible: `python -m pip install -c constraints.txt -e ".[dev]"`.
4. Ejecutar: `uvicorn src.main:app --reload --port 8010`.
5. Verificar: `ruff check . ; mypy src config ; pytest -q ; python -m build`.
   Integración con Redis real: `FUDIA_REDIS_INTEGRATION=1 REDIS_URL=redis://... pytest -q tests/test_redis_integration.py`.
6. Evals live del router en Lite y Flash, con credenciales del proveedor configurado: `python scripts/run_evals.py`.

## Estado

Despliegue GCP por push a `feat/ui-standard` mediante `foods-gcp`, con alternativa
manual: [runbook](../foods-infrastructure/docs/12_CONCIERGE_GCP.md) y workflow
reutilizable `fudia-concierge-gcp.yml`. Cambios exclusivos del bot no fuerzan
otros componentes al día ni migraciones. Una imagen
para HTTP, worker pool y Job de preflight. Redis privado con CA montada mediante
`REDIS_CA_CERTFILE` en HTTP o `REDIS_CA_CERT_PEM` en worker/preflight (sin volumen),
AUTH/TLS verificado y secretos versionados; Vertex usa ADC.
No se ha ejecutado ese despliegue ni el gate externo por agregar este código.

El alcance de código definido para Concierge está cerrado: QR por mesa y habilitación global por módulo, carta real, carrito, combos, modificadores, confirmación protegida, rondas independientes sobre una misma comanda, KDS, solicitud de cuenta con total real, handoff humano, multiagente, aislamiento multi-tenant, cola durable, concurrencia acotada con backpressure, retries, rate limiting, readiness, métricas, trazabilidad, evals y contenedor no-root.

La recuperación por etapas, el envío separado, el lock con fencing, el orden por conversación y la retención están implementados y cubiertos por pruebas con dobles y con un emulador de Redis (`docs/08_RECOVERY_AND_OPERATIONS.md`). Esas pruebas no sustituyen la validación real: siguen pendientes la integración contra Redis real (CI o staging), la confirmación de que Meta devuelve `biz_opaque_callback_data` en los estados y el **gate de activación externa** con credenciales reales de Meta/Vertex AI/Redis y una prueba de humo en el entorno desplegado. Ninguno se considera ejecutado mientras no se proporcionen esas credenciales y entorno.
