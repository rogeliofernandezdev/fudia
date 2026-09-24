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

- [ ] Número WhatsApp configurable por empresa.
- [ ] Botón Pedir por WhatsApp en página QR.
- [ ] Deeplink con token.
- [ ] Activar/desactivar Concierge por empresa/local.

## Bloque 4 — Conversación avanzada

- [ ] Combos/modificadores.
- [ ] Agregar a comanda ya abierta.
- [ ] Handoff humano.
- [ ] Observabilidad.
