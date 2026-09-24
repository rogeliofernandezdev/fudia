# Arquitectura

Fudia Concierge es el quinto proyecto desplegable del monorepo.

Cliente/WhatsApp -> FastAPI webhook -> ConciergeService -> LangGraph/OpenAI tools -> foods-backend.
Redis conserva sesión y carrito efímero.

## Fronteras

Concierge es responsable de conversación, intención, carrito temporal, presentación y confirmación.

foods-backend es responsable de tenencia, identidad del QR, menú, disponibilidad, precios, combos, stock, recetas, mesa, creación del pedido, KDS y auditoría.

No se replica el catálogo comercial en Concierge.
