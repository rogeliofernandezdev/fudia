# Fudia Concierge

Asistente conversacional de Fudia para que un comensal pueda iniciar un pedido desde el QR de su mesa y continuarlo por WhatsApp.

## Flujo inicial

QR de mesa -> WhatsApp -> Fudia Concierge -> herramientas -> foods-backend -> pedido confirmado -> Pedidos / KDS.

Fudia Concierge interpreta la conversación y mantiene el carrito temporal. foods-backend vuelve a validar producto, disponibilidad, precio, mesa y stock antes de registrar un pedido.

## Stack

- Python 3.12
- FastAPI
- LangGraph
- OpenAI Responses API con function tools
- WhatsApp Business Cloud API
- Redis para sesión conversacional efímera
- HTTP/OpenAPI para integración con foods-backend

## Desarrollo local

1. Copiar .env.example a .env.
2. Configurar OPENAI_API_KEY, REDIS_URL y las credenciales de WhatsApp.
3. Instalar: python -m pip install -e ".[dev]"
4. Ejecutar: uvicorn src.main:app --reload --port 8010
5. Verificar: ruff check . ; mypy src config ; pytest -q ; python -m build

## Estado

La primera vertical implementa webhook, QR, sesión, carrito, herramientas y confirmación protegida. Ver docs/05_IMPLEMENTATION_PLAN.md.
