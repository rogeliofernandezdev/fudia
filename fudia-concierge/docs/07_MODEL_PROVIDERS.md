# Proveedores de modelo y niveles

Concierge usa **Gemini en Vertex AI** por defecto (`LLM_PROVIDER=vertex`).
OpenAI se mantiene como alternativa seleccionable (`LLM_PROVIDER=openai`)
sobre el mismo contrato (`ConciergeRouter`, `ConversationAgent`) y los mismos
schemas de tools (`src/infrastructure/tool_schemas.py`).

## Niveles por dificultad

| Tarea | Nivel | Modelo por defecto | Motivo |
|---|---|---|---|
| Router (`out_of_scope/menu/order/service`) | Lite | `gemini-3.5-flash-lite` | Clasificación corta, salida restringida a cuatro etiquetas. |
| Carta (`menu`) | Flash | `gemini-3.5-flash` | Interpreta preguntas abiertas («¿qué tienes para hoy?», «algo ligero para dos»), saludos y referencias a lo ya mostrado; Lite las tomaba de forma literal. |
| Servicio (`service`) | Lite | `gemini-3.5-flash-lite` | Pedir la cuenta o al personal: una sola acción. |
| Pedido (`order`) | Flash | `gemini-3.5-flash` | Encadena búsqueda, modificadores, combos, carrito y confirmación. |

La política vive en `src/infrastructure/ai_factory.py` (`ROUTER_TIER`,
`AGENT_TIERS`) y los modelos en `GEMINI_MODEL_LITE` / `GEMINI_MODEL_FLASH`.
Se usan los modelos con disponibilidad mínima de 12 meses (retiro no antes de
mayo y julio de 2027). Las versiones de disponibilidad corta (por ejemplo
`gemini-3.8-flash`) solo deben configurarse tras evaluarlas y conociendo su
fecha de retiro: <https://docs.cloud.google.com/vertex-ai/generative-ai/docs/learn/model-versions>.

## Escalado de Lite a Flash

`TieredRouter` y `TieredAgent` (`src/ai/tiering.py`):

- Si el router Lite falla (error, timeout), Flash clasifica el mismo mensaje.
  Clasificar no tiene efectos secundarios.
- Si un agente Lite falla o termina sin respuesta útil (texto vacío o tope de
  vueltas de tools), Flash repite el turno **solo si el intento Lite no ejecutó
  ninguna tool con efectos** (`add_*`, `remove_item`, `prepare_confirmation`,
  `confirm_order`, `request_bill`, `request_human`). Repetir un turno que ya
  agregó productos o pidió la cuenta duplicaría la acción.
- Si no se puede escalar, se responde un texto seguro (`AgentIncompleteError`)
  o el error sube a la cola, que reintenta y, al agotar intentos, avisa al
  cliente (`FAILURE_REPLY`) y mueve el mensaje a dead-letter.

El agente de pedido ya usa Flash y no escala.

## Integración Vertex

- SDK `google-genai==2.27.0` (se mantiene `<3.0`, antes del cambio de
  function calling automático anunciado para la siguiente versión mayor).
- Cliente `genai.Client(vertexai=True, project, location)` con
  `HttpOptions(timeout)` y reintentos ante 408/429/5xx (`LLM_MAX_RETRIES`).
- Autenticación con **Application Default Credentials**: en Cloud Run, la
  cuenta de servicio del servicio; en local,
  `gcloud auth application-default login`. No hay API keys de Gemini.
- Router con salida restringida (`response_mime_type="text/x.enum"` y enum de
  rutas): una respuesta decorada no puede confundirse con una ruta; lo
  desconocido sigue siendo `out_of_scope`.
- Agentes con function calling **manual** (automático deshabilitado): toda
  tool pasa por `ToolRegistry`, que aplica la allowlist por intención. Cada
  llamada reenvía `system_instruction`, y el turno del modelo se devuelve
  intacto para conservar las *thought signatures* entre llamadas a tools.
- Límites: `LLM_TIMEOUT_SECONDS` por llamada, `LLM_MAX_OUTPUT_TOKENS` y
  `LLM_MAX_TOOL_ROUNDS` por turno.

## Configuración

| Variable | Defecto | Uso |
|---|---|---|
| `LLM_PROVIDER` | `vertex` | `vertex` u `openai`. |
| `GOOGLE_CLOUD_PROJECT` | — | Proyecto con Vertex AI habilitado. Obligatorio en producción con Vertex. |
| `GOOGLE_CLOUD_LOCATION` | `global` | Endpoint global o una región soportada por los modelos. |
| `GEMINI_MODEL_LITE` | `gemini-3.5-flash-lite` | Router, carta y servicio. |
| `GEMINI_MODEL_FLASH` | `gemini-3.5-flash` | Pedido y escalado. |
| `LLM_TIMEOUT_SECONDS` | `20` | Tiempo máximo por llamada al modelo. |
| `LLM_MAX_RETRIES` | `2` | Reintentos del SDK ante 408/429/5xx. |
| `LLM_MAX_OUTPUT_TOKENS` | `1024` | Tope de salida por llamada de agente. |
| `LLM_MAX_TOOL_ROUNDS` | `8` | Vueltas de tools por turno. |
| `OPENAI_API_KEY`, `OPENAI_MODEL`, `OPENAI_MODEL_LITE` | — | Solo con `LLM_PROVIDER=openai`. Sin `OPENAI_MODEL_LITE` todo usa `OPENAI_MODEL` y no hay escalado. |

## Requisitos en GCP

1. Habilitar la API de Vertex AI (`aiplatform.googleapis.com`) en el proyecto.
2. Cuenta de servicio propia para Concierge con `roles/aiplatform.user`
   (invocar modelos) y acceso a sus secretos en Secret Manager.
3. Cuotas de Gemini suficientes para la concurrencia del worker
   (`QUEUE_WORKER_CONCURRENCY`).

## Verificación

- Offline: `pytest -q` cubre el adaptador Vertex con un cliente falso
  (enum del router, instrucciones en cada vuelta, respuesta de tool con `id`,
  tope de vueltas, texto vacío), las reglas de escalado y el aviso al cliente.
- Live: `python scripts/run_evals.py` evalúa el router en **ambos niveles**
  (Lite y Flash) con los corpus de alcance, inyección e intención. Requiere
  `GOOGLE_CLOUD_PROJECT` y ADC.
- Pendiente con credenciales reales: confirmar que Vertex acepta los schemas
  de tools tal como están (`parameters_json_schema` con `additionalProperties`
  y `minimum`) y una prueba punta a punta QR → WhatsApp → pedido → KDS → cuenta.
