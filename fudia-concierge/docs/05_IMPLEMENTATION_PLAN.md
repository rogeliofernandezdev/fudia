# Plan de implementación

## Bloque 1 — Fundación

- [x] Proyecto en monorepo.
- [x] FastAPI y health.
- [x] Webhook WhatsApp.
- [x] Adaptador WhatsApp.
- [x] OpenAI Responses API con tools.
- [x] LangGraph.
- [x] Redis para sesión/carrito.
- [x] Cliente Fudia.
- [x] Confirmación explícita protegida por código.
- [x] Tests y CI.

## Bloque 2 — Contrato Fudia

- [x] Menú server-to-server derivado del QR.
- [x] Creación de pedido derivada del QR.
- [x] Revalidación de precio, disponibilidad y stock en foods-backend.
- [x] Integración KDS con canal WhatsApp.
- [x] Idempotencia por conversación ante reintentos.
- [x] Credencial propia para el contrato Concierge -> foods-backend.
- [x] Test QR -> pedido -> KDS.

## Bloque 3 — QR y configuración

- [x] Número WhatsApp configurable por empresa/local.
- [x] Botón Pedir por WhatsApp en página QR.
- [x] Deeplink con token.
- [x] Activar/desactivar Concierge por empresa/local.
- [x] El contrato server-to-server queda bloqueado cuando el local desactiva Concierge.
- [x] La sesión conversacional rechaza un QR de un local con Concierge desactivado.

## Bloque 4 — Conversación avanzada

- [ ] Combos conversacionales con grupos y selecciones validadas por Fudia.
- [ ] Modificadores de productos simples.
- [ ] Agregar a comanda ya abierta.
- [ ] Handoff humano.
- [ ] Observabilidad.
