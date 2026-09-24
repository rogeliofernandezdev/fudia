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
- [x] Idempotencia por ronda con conversationId estable y requestId independiente.
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

- [x] Combos conversacionales con grupos y selecciones validadas por Fudia.
- [x] Modificadores de productos simples.
- [x] Agregar nuevas rondas a una misma comanda aunque rondas anteriores estén preparando o listas.
- [x] KDS independiente por ronda sin reprocesar productos anteriores.
- [x] Solicitud de cuenta con consumo acumulado, pagos y saldo desde foods-backend.
- [x] Handoff humano persistente y resoluble desde Operaciones.
- [x] Observabilidad estructurada con privacidad.
- [x] Bandeja operativa de pedidos conectada a datos reales.
- [x] Aislamiento por número receptor de WhatsApp.
- [x] Módulo `whatsapp_bot` listo para activación por entitlement.

## Bloque 5 — Despliegue y endurecimiento externo

- [ ] Validar webhook real de Meta con firma y varios números conectados.
- [ ] Validar envío real con el `phone_number_id` receptor.
- [ ] Configurar secretos de OpenAI, Meta, Redis y Fudia en el entorno.
- [ ] Crear alertas operativas a partir de los eventos estructurados.
- [ ] Ejecutar prueba de humo QR -> WhatsApp -> múltiples rondas -> KDS -> cuenta en producción controlada.
