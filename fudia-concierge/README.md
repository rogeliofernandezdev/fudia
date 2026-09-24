# Fudia Concierge

Asistente conversacional de Fudia para atender por WhatsApp a un comensal sentado en una mesa, desde el escaneo del QR hasta la solicitud de la cuenta.

## Flujo

QR de mesa -> WhatsApp -> Fudia Concierge -> foods-backend -> carta/disponibilidad -> ronda confirmada -> KDS -> nuevas rondas -> cuenta acumulada.

Fudia Concierge mantiene únicamente el carrito temporal de la ronda en curso. foods-backend revalida producto, disponibilidad, precio, mesa y stock antes de registrar cada ronda y conserva la comanda acumulada como fuente de verdad.

Cuando el cliente solicita la cuenta, Concierge consulta el consumo real de la mesa en backend y muestra productos, total, pagos registrados y saldo pendiente. Solicitar la cuenta no registra un pago; el cobro continúa por Caja/POS.

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

El alcance funcional incluye QR y configuración por local, carta real por nombre/descripción/categoría, carrito, combos, modificadores, confirmación protegida, rondas independientes sobre una misma comanda, KDS, solicitud de cuenta con total real, handoff humano y observabilidad.

Queda como validación de despliegue la prueba contra credenciales reales de Meta/OpenAI/Redis.