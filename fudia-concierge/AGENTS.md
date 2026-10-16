# AGENTS — Fudia Concierge

## Objetivo

Implementar la experiencia conversacional del comensal para Fudia. El canal
del alcance actual es WhatsApp iniciado desde el QR de una mesa; al terminar la
atención en mesa, el mismo chat toma pedidos de recojo o delivery del último
local visitado por QR.

## Reglas

1. foods-backend es la única fuente de verdad de productos, disponibilidad, precios y pedidos.
2. El modelo nunca inventa productos, precios, promociones ni disponibilidad.
3. Toda mutación irreversible se ejecuta mediante una herramienta determinista.
4. Crear un pedido exige una confirmación explícita del cliente.
5. El QR determina empresa, local y mesa; nunca se aceptan esos alcances desde texto libre generado por el modelo.
6. No persistir secretos, tokens de Meta ni claves de modelos en el repositorio. Vertex AI usa Application Default Credentials, nunca API keys.
7. Los prompts viven en prompts/ y se versionan.
8. La conversación puede ser efímera; el pedido confirmado siempre se persiste en foods-backend.
9. Mantener adaptadores separados para cada proveedor de modelo (Vertex AI, OpenAI), WhatsApp, Redis y Fudia. La política de niveles (Flash-Lite/Flash) y el escalado viven en `src/ai/tiering.py` y `src/infrastructure/ai_factory.py`; nunca se escala un turno que ya ejecutó una tool con efectos.
10. Añadir pruebas para cada herramienta que pueda cambiar el carrito o crear pedidos.
11. No copiar lógica contable, OCR ni SUNAT desde tuconta-assistant.
12. WhatsApp es un canal global de Fudia: un solo bot. El número oficial es propiedad de la configuración de Meta / WhatsApp Business y no debe duplicarse como configuración de Fudia. El uso de Concierge depende de `whatsapp_bot`, activable únicamente desde plataforma.
13. Ejecutar ruff, mypy, pytest y build antes de cerrar cambios.
14. Toda tool con efectos registra su punto de recuperación mediante el `EffectJournal` (`docs/08_RECOVERY_AND_OPERATIONS.md`): un efecto local con `local_effect`; una operación del backend con `before_effect` (clave idempotente y datos de reintento) y `after_effect`/`effect_failed`. Nunca escribir sesión o registro fuera de `ConversationStore.commit`.
