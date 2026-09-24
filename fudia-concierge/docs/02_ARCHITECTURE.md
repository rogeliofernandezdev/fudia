# Arquitectura

Fudia Concierge es un servicio desplegable del monorepo Fudia.

Cliente/WhatsApp -> FastAPI webhook -> ConciergeService -> LangGraph/OpenAI tools -> foods-backend.

Redis conserva sesión, contexto conversacional y carrito efímero de la ronda en curso.

## Fronteras

Concierge es responsable de conversación, intención, carrito temporal, presentación, confirmación de cada ronda y solicitud de cuenta.

foods-backend es responsable de tenencia, identidad del QR, menú, categorías, disponibilidad, precios, combos, stock, recetas, mesa, comanda acumulada, rondas de cocina, pagos, saldo, KDS y auditoría.

Una conversación de WhatsApp mantiene un conversationId estable. Cada ronda confirmada usa un requestId distinto como clave idempotente. No se replica el catálogo comercial ni se calcula la cuenta desde memoria conversacional.

## Separación de prompts

Los prompts tienen una sola responsabilidad cada uno:

- `prompts/scope_router.md` decide únicamente si el mensaje pertenece al servicio de la mesa.
- `prompts/system.md` gobierna la conversación y el uso de herramientas después de superar ese filtro.

Las reglas de negocio, validaciones de stock, precios, confirmación e idempotencia permanecen en código y foods-backend; no se delegan al prompt.
